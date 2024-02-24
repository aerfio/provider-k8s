package object

import (
	"context"
	"fmt"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	objv1alpha1 "aerf.io/provider-k8s/apis/object/v1alpha1"
	apisv1alpha1 "aerf.io/provider-k8s/apis/v1alpha1"
	"aerf.io/provider-k8s/internal/cacheregistry"
	"aerf.io/provider-k8s/internal/celcheck"
	"aerf.io/provider-k8s/internal/controllers/generic"
	"aerf.io/provider-k8s/internal/k8scmp"
	"aerf.io/provider-k8s/internal/restcfgutil"
)

const (
	errNotObject    = "managed resource is not a Object custom resource"
	errTrackPCUsage = "cannot track ProviderConfig"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

// Setup adds a controller that reconciles Object managed resources.
func Setup(mgr ctrl.Manager, o controller.Options, registry *cacheregistry.Registry) error {
	name := managed.ControllerName(objv1alpha1.ObjectGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnecter(&connector{
			client:       mgr.GetClient(),
			usageTracker: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			logger:       o.Logger,
			registry:     registry,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithCreationGracePeriod(3 * time.Second),
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(objv1alpha1.ObjectGroupVersionKind), opts...)

	objectController, err := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&objv1alpha1.Object{}).
		Build(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
	if err != nil {
		return err
	}
	registry.SetRegisterFn(func(inf cache.Informer, parentNameNs types.NamespacedName) error {
		return objectController.Watch(&source.Informer{Informer: inf}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, c client.Object) []reconcile.Request {
			o.Logger.WithValues("name", "object-controller-watch", "objectRef", meta.TypedReferenceTo(c, c.GetObjectKind().GroupVersionKind()), "parentRef", parentNameNs).Debug("enqueuing reconcile request")
			return []reconcile.Request{
				{
					NamespacedName: parentNameNs,
				},
			}
		}))
	})
	return nil
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	client       client.Client
	usageTracker resource.Tracker
	logger       logging.Logger
	registry     *cacheregistry.Registry
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

	rc, err := restcfgutil.RestConfigFromProviderConfig(ctx, pc, c.client)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	remoteCli, err := client.New(rc, client.Options{})
	if err != nil {
		return nil, err
	}

	return generic.NewExternalForType[*objv1alpha1.Object](&external{
		localCli:      c.client,
		remoteCli:     remoteCli,
		log:           c.logger,
		registry:      c.registry,
		remoteRestCfg: rc,
	}, errors.New(errNotObject)), nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	localCli      client.Client
	remoteCli     client.Client
	log           logging.Logger
	registry      *cacheregistry.Registry
	remoteRestCfg *rest.Config
}

func (e *external) Observe(ctx context.Context, cr *objv1alpha1.Object) (managed.ExternalObservation, error) {
	log := e.loggerFor(cr)
	log.Debug("Observing", "reconciledObject", cr)

	desired, err := cr.GetDesired()
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if meta.WasDeleted(cr) {
		err = e.registry.StopAndRemove(e.remoteRestCfg, desired.GroupVersionKind(), client.ObjectKeyFromObject(desired))
	} else {
		err = e.registry.RegisterCacheFromRestConfig(e.remoteRestCfg, desired.GroupVersionKind(), client.ObjectKeyFromObject(desired), client.ObjectKeyFromObject(cr))
	}
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

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: !e.hasDrifted(observed, desired),
		Diff:             k8scmp.DiffUnstructured(observed, desired),
	}, errors.Wrap(e.setObserved(cr, observed), "failed to derive object status from the observed remote object")
}

func (e *external) Create(ctx context.Context, cr *objv1alpha1.Object) (managed.ExternalCreation, error) {
	log := e.loggerFor(cr)
	log.Debug("Creating")

	desired, err := cr.GetDesired()
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if err := e.Apply(ctx, desired); err != nil {
		return managed.ExternalCreation{}, err
	}

	log.Debug("Created object", "object", desired)

	return managed.ExternalCreation{}, errors.Wrap(e.setObserved(cr, desired), "failed to derive object status from the observed remote object")
}

func (e *external) Update(ctx context.Context, cr *objv1alpha1.Object) (managed.ExternalUpdate, error) {
	e.loggerFor(cr).Debug("Updating")

	desired, err := cr.GetDesired()
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.Apply(ctx, desired); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, e.updateConditionFromObserved(cr, desired)
}

func (e *external) Delete(ctx context.Context, cr *objv1alpha1.Object) error {
	e.loggerFor(cr).Debug("Deleting")

	desired, err := cr.GetDesired()
	if err != nil {
		return err
	}

	if err := e.registry.StopAndRemove(e.remoteRestCfg, desired.GroupVersionKind(), client.ObjectKeyFromObject(desired)); err != nil {
		return errors.Wrapf(err, "failed to stop the cache for cluster with host url %q, object gvk %q, name/ns %q", e.remoteRestCfg.Host, desired.GroupVersionKind(), client.ObjectKeyFromObject(desired))
	}

	return errors.Wrap(client.IgnoreNotFound(e.remoteCli.Delete(ctx, desired)), "failed to delete external object")
}

func (e *external) loggerFor(obj client.Object) logging.Logger {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return e.log.WithValues("name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", gvk.Kind, "group", gvk.Group, "version", gvk.Version)
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
	case objv1alpha1.ReadinessPolicyUseCELExpression:
		ready, err := celcheck.Eval(obj.Spec.Readiness.CELExpression, observed.UnstructuredContent())
		if err != nil {
			return errors.Wrap(err, "failed to run CEL expression on observed object")
		}
		cond := xpv1.Unavailable()
		if ready {
			cond = xpv1.Available()
		}
		obj.SetConditions(cond)

		return nil
	default:
		// should never happen
		return errors.Errorf("unknown readiness policy %q", obj.Spec.Readiness.Policy)
	}
	return nil
}

func (e *external) setObserved(obj *objv1alpha1.Object, observed *unstructured.Unstructured) error {
	var err error
	if obj.Status.AtProvider.Manifest.Raw, err = observed.MarshalJSON(); err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	if err := e.updateConditionFromObserved(obj, observed); err != nil {
		return err
	}
	return nil
}
