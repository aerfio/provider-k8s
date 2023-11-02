/*
Copyright 2020 The Crossplane Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package object

import (
	"context"
	"encoding/json"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"

	objv1alpha1 "aerf.io/provider-k8s/apis/object/v1alpha1"
	apisv1alpha1 "aerf.io/provider-k8s/apis/v1alpha1"
	"aerf.io/provider-k8s/internal/controller/generic"
	"aerf.io/provider-k8s/internal/manageddiff"
)

const (
	errNotObject    = "managed resource is not a Object custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usageTracker"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

// Setup adds a controller that reconciles Object managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) (ctrlcontroller.Controller, error) {
	name := managed.ControllerName(objv1alpha1.ObjectGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnecter(&connector{
			client:       mgr.GetClient(),
			usageTracker: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			logger:       o.Logger,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(objv1alpha1.ObjectGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&objv1alpha1.Object{}).
		Build(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	client       client.Client
	usageTracker resource.Tracker
	logger       logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*objv1alpha1.Object)
	if !ok {
		return nil, errors.New(errNotObject)
	}

	if err := c.usageTracker.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.client.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	var err error
	var rc *rest.Config
	switch cd := pc.Spec.Credentials; cd.Source {
	case xpv1.CredentialsSourceInjectedIdentity:
		rc, err = ctrl.GetConfig()
		if err != nil {
			return nil, errors.Wrap(err, "couldn't get rest.Config from in-cluster data")
		}
	default:
		data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.client, cd.CommonCredentialSelectors)
		if err != nil {
			return nil, errors.Wrap(err, errGetCreds)
		}
		cfg, err := clientcmd.NewClientConfigFromBytes(data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create clientConfig from raw bytes")
		}

		rc, err = cfg.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create *rest.Config from kubeconfig: %s", err)
		}
	}

	remoteCli, err := client.New(rc, client.Options{})
	if err != nil {
		return nil, err
	}

	return generic.NewExternalForType[*objv1alpha1.Object](&external{
		localCli:  c.client,
		remoteCli: remoteCli,
		log:       c.logger,
	}, errors.New(errNotObject)), nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	localCli  client.Client
	remoteCli client.Client
	log       logging.Logger
}

func (e *external) Observe(ctx context.Context, cr *objv1alpha1.Object) (managed.ExternalObservation, error) {
	log := e.loggerFor(cr)
	log.Debug("Observing", "reconciledObject", cr)

	desired, err := e.getDesired(cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	observed := desired.DeepCopy()
	err = e.localCli.Get(ctx, types.NamespacedName{
		Namespace: desired.GetNamespace(),
		Name:      desired.GetName(),
	}, observed)
	if apierrors.IsNotFound(err) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	} else if err != nil {
		return managed.ExternalObservation{}, err
	}

	if err := e.ApplyDryRun(ctx, desired); err != nil {
		return managed.ExternalObservation{}, err
	}

	resourceUpToDate := !e.hasDrifted(observed, desired)

	if resourceUpToDate {
		cr.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: resourceUpToDate,
		Diff:             manageddiff.SafeDiff(observed, desired),
	}, nil
}

func (e *external) Create(ctx context.Context, cr *objv1alpha1.Object) (managed.ExternalCreation, error) {
	log := e.loggerFor(cr)
	log.Debug("Creating")

	desired, err := e.getDesired(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := e.Apply(ctx, desired); err != nil {
		return managed.ExternalCreation{}, err
	}

	log.Debug("Created object", "object", desired)

	return managed.ExternalCreation{}, e.updateConditionFromObserved(cr, desired)
}

func (e *external) Update(ctx context.Context, cr *objv1alpha1.Object) (managed.ExternalUpdate, error) {
	e.loggerFor(cr).Debug("Updating")

	desired, err := e.getDesired(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	desiredCopy := desired.DeepCopy()
	if err := e.Apply(ctx, desired); err != nil {
		return managed.ExternalUpdate{}, err
	}
	if diff := manageddiff.SafeDiff(desiredCopy, desired); diff != "" {
		e.loggerFor(cr).Debug("Difference between the reconciled object", "diff", diff)
	}

	return managed.ExternalUpdate{}, e.updateConditionFromObserved(cr, desired)
}

func (e *external) Delete(ctx context.Context, cr *objv1alpha1.Object) error {
	e.loggerFor(cr).Debug("Deleting")

	desired, err := e.getDesired(cr)
	if err != nil {
		return err
	}

	err = e.remoteCli.Delete(ctx, desired)
	if apierrors.IsNotFound(err) {
		return nil
	}

	return err
}

func (e *external) loggerFor(obj client.Object) logging.Logger {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return e.log.WithValues("name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", gvk.Kind, "group", gvk.Group, "version", gvk.Version)
}

func (e *external) getDesired(obj *objv1alpha1.Object) (*unstructured.Unstructured, error) {
	desired := &unstructured.Unstructured{}
	if err := json.Unmarshal(obj.Spec.ForProvider.Manifest.Raw, desired); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal raw manifest")
	}

	return desired, nil
}

func (e *external) Apply(ctx context.Context, obj client.Object, opts ...client.PatchOption) error {
	patchOpts := append(opts, client.ForceOwnership, client.FieldOwner("provider-k8s")) // nolint:gocritic // it's deliberate
	return e.remoteCli.Patch(ctx, obj, client.Apply, patchOpts...)
}

func (e *external) ApplyDryRun(ctx context.Context, obj client.Object) error {
	return e.Apply(ctx, obj, client.DryRunAll)
}

func (e *external) updateConditionFromObserved(obj *objv1alpha1.Object, observed *unstructured.Unstructured) error {
	log := e.loggerFor(obj)
	switch obj.Spec.Readiness.Policy {
	case objv1alpha1.ReadinessPolicyDeriveFromObject:
		conditioned := xpv1.ConditionedStatus{}
		err := fieldpath.Pave(observed.Object).GetValueInto("status", &conditioned)
		if err != nil {
			log.Debug("Got error while getting conditions from observed object, setting it as Unavailable", "error", err, "observed", observed)
			obj.SetConditions(xpv1.Unavailable().WithMessage("Got error while getting conditions from observed object"))
			return errors.Wrap(err, "failed to get conditions from observed object")
		}
		if status := conditioned.GetCondition(xpv1.TypeReady).Status; status != corev1.ConditionTrue {
			log.Debug("Observed object is not ready, setting it as Unavailable", "status", status, "observed", observed)
			obj.SetConditions(xpv1.Unavailable().WithMessage(fmt.Sprintf("Observed object's condition with type %q is %q but should be %q", xpv1.TypeReady, status, corev1.ConditionTrue)))
			return nil
		}
		obsrvdGenStatus := objv1alpha1.StatusWithObservedGeneration{}
		if err := fieldpath.Pave(observed.Object).GetValueInto("status", &obsrvdGenStatus); err == nil {
			if observed.GetGeneration() != obsrvdGenStatus.ObservedGeneration {
				log.Debug("Observed object is not ready, setting it as Unavailable", "status", obsrvdGenStatus, "observed", observed)
				obj.SetConditions(xpv1.Unavailable().WithMessage("Observed object's status.observedGeneration is not equal to metadata.generation"))
				return nil
			}
		}

		obj.SetConditions(xpv1.Available())
	case objv1alpha1.ReadinessPolicySuccessfulCreate, "":
		obj.SetConditions(xpv1.Available())
	default:
		// should never happen
		return errors.Errorf("unknown readiness policy %q", obj.Spec.Readiness.Policy)
	}
	return nil
}
