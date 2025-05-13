package template

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/notification/validation"
	"lightiot/pkg/util/security"
)

type REST struct {
	m *Manager
}

var _ rest.Interface = (*REST)(nil)
var _ rest.UpdateStrategy = (*REST)(nil)

func NewREST(mgr *Manager) *REST {
	return &REST{m: mgr}
}

func (r *REST) New() gruntime.Object {
	return &v1.Template{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return r.m.CreateTmpl(obj.(*v1.Template))
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	return r.m.GetTmplById(id)
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	filter := Filter{Tenant: security.GetTenant()}
	if err := mapstructure.Decode(options.Filter, &filter); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	rts, _ := r.m.ListTmpls(filter)
	return &runtime.ResponseModel{Templates: rts}, nil
}

func (r *REST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	// TODO
	return nil, nil
}

func (r *REST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return r.m.DeleteTmpl(id, options.Version)
}

func (r *REST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateTemplate(obj.(*v1.Template))
}

func (r *REST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	// TODO
	return nil
}
