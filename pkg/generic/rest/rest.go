package rest

import (
	"context"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/runtime"
)

type Newer interface {
	New() runtime.Object
}

type Getter interface {
	Get(ctx context.Context, id string, options *meta.GetOptions) (runtime.Object, error)
}

type ValidateObjectFunc func(ctx context.Context, obj runtime.Object) error

type Creater interface {
	Newer
	Create(ctx context.Context, obj runtime.Object, createValidation ValidateObjectFunc, options *meta.CreateOptions) (runtime.Object, error)
}

type ValidateObjectUpdateFunc func(ctx context.Context, obj, old runtime.Object) error

type Updater interface {
	Newer
	Update(ctx context.Context, id string, obj runtime.Object, updateValidation ValidateObjectUpdateFunc, options *meta.UpdateOptions) (runtime.Object, error)
}

type Patcher interface {
	Getter
	Updater
}

type Deleter interface {
	Delete(ctx context.Context, id string, options *meta.DeleteOptions) (runtime.Object, error)
}

type Lister interface {
	List(ctx context.Context, options *meta.ListOptions) (runtime.Object, error)
}

type Interface interface {
	Creater
	Lister
	Getter
	Updater
	Deleter
}
