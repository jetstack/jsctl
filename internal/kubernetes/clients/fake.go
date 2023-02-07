package clients

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type FakeGeneric[T, ListT runtime.Object] struct {
	FakeGet     func(context.Context, *GenericRequestOptions, T) error
	FakeList    func(context.Context, *GenericRequestOptions, ListT) error
	FakePresent func(context.Context, *GenericRequestOptions) (bool, error)
	FakePatch   func(context.Context, *GenericRequestOptions, []byte) error
}

var _ Generic[*runtime.Unknown, *runtime.Unknown] = &FakeGeneric[*runtime.Unknown, *runtime.Unknown]{}

func (f *FakeGeneric[T, ListT]) Get(ctx context.Context, options *GenericRequestOptions, result T) error {
	return f.FakeGet(ctx, options, result)
}

func (f *FakeGeneric[T, ListT]) List(ctx context.Context, options *GenericRequestOptions, result ListT) error {
	return f.FakeList(ctx, options, result)
}

func (f *FakeGeneric[T, ListT]) Present(ctx context.Context, options *GenericRequestOptions) (bool, error) {
	return f.FakePresent(ctx, options)
}
func (f *FakeGeneric[T, ListT]) Patch(ctx context.Context, options *GenericRequestOptions, patch []byte) error {
	return f.FakePatch(ctx, options, patch)
}
