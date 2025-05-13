package message

import (
	"context"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"lightiot/pkg/generic/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/notification/validation"
)

type REST struct {
	m *Manager
}

var (
	_ rest.Creater        = (*REST)(nil)
	_ rest.CreateStrategy = (*REST)(nil)
)

func NewREST(mgr *Manager) *REST {
	return &REST{m: mgr}
}

func (r *REST) New() runtime.Object {
	return &v1.Message{}
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (runtime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	err := r.m.SendMessage(obj.(*v1.Message))
	return nil, err
}

func (r *REST) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateMessage(obj.(*v1.Message))
}
