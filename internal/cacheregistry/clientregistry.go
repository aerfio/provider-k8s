package cacheregistry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cacheWithStopper struct {
	c        cache.Cache
	cancelFn context.CancelFunc
}

type Registry struct {
	mu         sync.Mutex
	cacheMap   map[cacheMapKey]cacheWithStopper
	log        logging.Logger
	registerFn func(cache.Informer, types.NamespacedName) error
}

func New(log logging.Logger) *Registry {
	return &Registry{
		cacheMap: make(map[cacheMapKey]cacheWithStopper),
		log:      log,
	}
}

func (r *Registry) SetRegisterFn(fn func(cache.Informer, types.NamespacedName) error) {
	r.registerFn = fn
}

type GVKWithNameNamespace struct {
	schema.GroupVersionKind
	types.NamespacedName
}

type cacheMapKey struct {
	GVKWithNameNamespace
	HostURL          string
	VersionedAPIPath string
}

func (r *Registry) getCacheWithStopper(key cacheMapKey) (cacheWithStopper, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	val, ok := r.cacheMap[key]
	return val, ok
}

func (r *Registry) setCacheWithStopper(key cacheMapKey, val cacheWithStopper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheMap[key] = val
}

func (r *Registry) RegisterCacheFromRestConfig(restCfg *rest.Config, gvk schema.GroupVersionKind, nameNs, parentNameNs types.NamespacedName) (retErr error) {
	hostURL, versionedAPIPath, err := rest.DefaultServerUrlFor(restCfg)
	if err != nil {
		return err
	}
	log := r.log.WithValues("name", nameNs.Name, "namespace", nameNs.Namespace, "gvk", gvk, "hostURL", hostURL, "versionedAPIPath", versionedAPIPath)
	key := cacheMapKey{
		GVKWithNameNamespace: GVKWithNameNamespace{
			GroupVersionKind: gvk,
			NamespacedName:   nameNs,
		},
		HostURL:          hostURL.String(),
		VersionedAPIPath: versionedAPIPath,
	}

	if _, ok := r.getCacheWithStopper(key); ok {
		log.Debug("cache already in registry")
		return nil
	}

	unstr := &unstructured.Unstructured{}
	unstr.SetGroupVersionKind(gvk)
	{
		cli, err := client.New(restCfg, client.Options{})
		if err != nil {
			return err
		}
		namespaced, err := cli.IsObjectNamespaced(unstr)
		if err != nil {
			return err
		}

		if nameNs.Namespace == "" && namespaced {
			nameNs.Namespace = "default"
		}
		cli = nil
	}

	nameFieldSelector := fields.OneTermEqualSelector("metadata.name", nameNs.Name)
	cacheByObj := cache.ByObject{}
	if nameNs.Namespace != "" {
		cacheByObj.Namespaces = map[string]cache.Config{
			nameNs.Namespace: {
				FieldSelector: nameFieldSelector,
			},
		}
	} else {
		cacheByObj.Field = nameFieldSelector
	}

	c, err := cache.New(restCfg, cache.Options{
		ReaderFailOnMissingInformer: true,
		ByObject: map[client.Object]cache.ByObject{
			unstr: cacheByObj,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create new cache: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.setCacheWithStopper(key, cacheWithStopper{
		c:        c,
		cancelFn: cancel,
	})
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, r.StopAndRemove(restCfg, gvk, nameNs))
		}
	}()

	go func() {
		log.Debug("starting cache")
		if err := c.Start(ctx); err != nil {
			log.Info(fmt.Sprintf("failed to run cache: %s", err))
		}
	}()

	syncCtx, syncCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer syncCancel()
	if !c.WaitForCacheSync(syncCtx) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// ctx background cause informers are already started
	inf, err := c.GetInformerForKind(context.Background(), gvk)
	if err != nil {
		kindMatchErr := &meta.NoKindMatchError{}
		switch {
		case errors.As(err, &kindMatchErr):
			return fmt.Errorf("CRD seems to not be installed yet, groupKind: %s, error: %s", kindMatchErr.GroupKind, err)
		case runtime.IsNotRegisteredError(err):
			return fmt.Errorf("GVK must be registered to the Scheme: %s", err)
		default:
			return fmt.Errorf("failed to get informer for %q: %s", gvk.String(), err)
		}
	}

	if err := r.registerFn(inf, parentNameNs); err != nil {
		return err
	}
	return nil
}

func (r *Registry) StopAndRemove(restCfg *rest.Config, gvk schema.GroupVersionKind, nameNs types.NamespacedName) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hostURL, versionedAPIPath, err := rest.DefaultServerUrlFor(restCfg)
	if err != nil {
		return err
	}
	key := cacheMapKey{
		GVKWithNameNamespace: GVKWithNameNamespace{
			GroupVersionKind: gvk,
			NamespacedName:   nameNs,
		},
		HostURL:          hostURL.String(),
		VersionedAPIPath: versionedAPIPath,
	}
	c, ok := r.cacheMap[key]
	if !ok {
		return nil
	}

	r.log.Debug("stopping cache", "key", key)
	c.cancelFn()
	delete(r.cacheMap, key)
	return nil
}
