package cacheregistry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"go.uber.org/atomic"
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

func (r *Registry) RegisterCacheFromRestConfig(restCfg *rest.Config, gvk schema.GroupVersionKind, nameNs, parentNameNs types.NamespacedName) error {
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

	if _, ok := r.cacheMap[key]; ok {
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
		return err
	}

	scc := &startCountingCache{
		log:          r.log,
		c:            c,
		startedTimes: atomic.NewInt32(0),
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cacheMap[key] = cacheWithStopper{
		c:        scc,
		cancelFn: cancel,
	}
	r.mu.Unlock()

	go func() {
		log.Debug("starting cache")
		if err := scc.Start(ctx); err != nil {
			log.Info(fmt.Sprintf("failed to run cache: %s", err))
		}
	}()

	syncCtx, syncCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer syncCancel()
	if !scc.WaitForCacheSync(syncCtx) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// ctx background cause informers are already started
	inf, err := scc.GetInformerForKind(context.Background(), gvk)
	if err != nil {
		cancel()
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
		cancel()
		return err
	}
	return nil
}

type startCountingCache struct {
	log          logging.Logger
	c            cache.Cache
	startedTimes *atomic.Int32
}

func (s *startCountingCache) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return s.c.Get(ctx, key, obj, opts...)
}

func (s *startCountingCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return s.c.List(ctx, list, opts...)
}

func (s *startCountingCache) GetInformer(ctx context.Context, obj client.Object, opts ...cache.InformerGetOption) (cache.Informer, error) {
	return s.c.GetInformer(ctx, obj, opts...)
}

func (s *startCountingCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind, opts ...cache.InformerGetOption) (cache.Informer, error) {
	return s.c.GetInformerForKind(ctx, gvk, opts...)
}

func (s *startCountingCache) Start(ctx context.Context) error {
	s.startedTimes.Inc()
	s.log.Info("started cache", "count", s.startedTimes.Load())
	return s.c.Start(ctx)
}

func (s *startCountingCache) WaitForCacheSync(ctx context.Context) bool {
	return s.c.WaitForCacheSync(ctx)
}

func (s *startCountingCache) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	return s.c.IndexField(ctx, obj, field, extractValue)
}

var _ cache.Cache = &startCountingCache{}

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
	c.c = nil
	delete(r.cacheMap, key)
	return nil
}
