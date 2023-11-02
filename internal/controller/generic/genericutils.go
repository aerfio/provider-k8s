package generic

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

type ObservedGenerationSetter interface {
	SetObservedGeneration(int64)
}

type GenericExternalClient[T resource.Managed] interface {
	// Observe the external resource the supplied Managed resource
	// represents, if any. Observe implementations must not modify the
	// external resource, but may update the supplied Managed resource to
	// reflect the state of the external resource. Status modifications are
	// automatically persisted unless ResourceLateInitialized is true - see
	// ResourceLateInitialized for more detail.
	Observe(ctx context.Context, mg T) (managed.ExternalObservation, error)

	// Create an external resource per the specifications of the supplied
	// Managed resource. Called when Observe reports that the associated
	// external resource does not exist. Create implementations may update
	// managed resource annotations, and those updates will be persisted.
	// All other updates will be discarded.
	Create(ctx context.Context, mg T) (managed.ExternalCreation, error)

	// Update the external resource represented by the supplied Managed
	// resource, if necessary. Called unless Observe reports that the
	// associated external resource is up to date.
	Update(ctx context.Context, mg T) (managed.ExternalUpdate, error)

	// Delete the external resource upon deletion of its associated Managed
	// resource. Called when the managed resource has been deleted.
	Delete(ctx context.Context, mg T) error
}

func NewExternalForType[T resource.Managed](specificExternal GenericExternalClient[T], errNotConfiguredType error) *GeneralExternal[T] {
	return &GeneralExternal[T]{
		genericExternal:      specificExternal,
		errNotConfiguredType: errNotConfiguredType,
	}
}

type GeneralExternal[T resource.Managed] struct {
	genericExternal      GenericExternalClient[T]
	errNotConfiguredType error
}

func (g *GeneralExternal[T]) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(T)
	if !ok {
		return managed.ExternalObservation{}, g.errNotConfiguredType
	}

	if setter, ok := mg.(ObservedGenerationSetter); ok {
		setter.SetObservedGeneration(mg.GetGeneration())
	}

	return g.genericExternal.Observe(ctx, cr)
}

func (g *GeneralExternal[T]) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(T)
	if !ok {
		return managed.ExternalCreation{}, g.errNotConfiguredType
	}
	if setter, ok := mg.(ObservedGenerationSetter); ok {
		setter.SetObservedGeneration(mg.GetGeneration())
	}

	return g.genericExternal.Create(ctx, cr)
}

func (g *GeneralExternal[T]) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(T)
	if !ok {
		return managed.ExternalUpdate{}, g.errNotConfiguredType
	}
	if setter, ok := mg.(ObservedGenerationSetter); ok {
		setter.SetObservedGeneration(mg.GetGeneration())
	}

	return g.genericExternal.Update(ctx, cr)
}

func (g *GeneralExternal[T]) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(T)
	if !ok {
		return g.errNotConfiguredType
	}

	if setter, ok := mg.(ObservedGenerationSetter); ok {
		setter.SetObservedGeneration(mg.GetGeneration())
	}

	return g.genericExternal.Delete(ctx, cr)
}
