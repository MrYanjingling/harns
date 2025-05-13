package propertysettype

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"lightiot/pkg/model/validation"
	"lightiot/pkg/util/security"
)

type REST struct {
	m *Manager
}

var _ rest.Interface = (*REST)(nil)
var _ rest.UpdateStrategy = (*REST)(nil)

func (r *REST) New() gruntime.Object {
	return &v1.PropertySetType{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return r.m.CreatePropertySetType(obj.(*v1.PropertySetType))
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	return r.m.GetPropertySetTypeById(id)
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	f := filter{Tenant: security.GetTenant()}
	if err := mapstructure.Decode(options.Filter, &f); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	psts, _ := r.m.GetPropertySetTypes(&f)
	return &runtime.ResponseModel{PropertySetTypes: psts}, nil
}

func (r *REST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	pst, err := r.m.GetPropertySetTypeById(id)
	if err != nil {
		return nil, err
	}
	copied := pst.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return r.m.updatePropertySetType(id, options.Version, obj.(*v1.PropertySetType), copied.(*runtime.PropertySetType))
}

func (r *REST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return r.m.deletePropertySetType(id, options.Version)
}

func (r *REST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidatePropertySetType(obj.(*v1.PropertySetType))
}

func (r *REST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidatePropertySetTypeUpdate(obj.(*v1.PropertySetType), old.(*runtime.PropertySetType))
}
