package thing

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
	"strconv"
)

type REST struct {
	m *Manager
}

var _ rest.Interface = (*REST)(nil)
var _ rest.UpdateStrategy = (*REST)(nil)

func (r *REST) New() gruntime.Object {
	return &v1.Thing{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return r.m.CreateThing(obj.(*v1.Thing))
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	return r.m.GetThingById(id)
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	filter := thingFilter{Tenant: security.GetTenant()}
	if err := mapstructure.Decode(options.Filter, &filter); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	things, _ := r.m.GetThings(&filter)
	return &runtime.ResponseModel{Things: things}, nil
}

func (r *REST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	t, err := r.m.GetThingById(id)
	if err != nil {
		return nil, err
	}
	copied := t.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return r.m.updateThing(id, options.Version, obj.(*v1.Thing), copied.(*runtime.Thing))
}

func (r *REST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return r.m.deleteThing(id, options.Version)
}

func (r *REST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateThing(obj.(*v1.Thing))
}

func (r *REST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidateThingUpdate(obj.(*v1.Thing), old.(*runtime.Thing))
}

type CharREST struct {
	m *Manager
}

func (cr *CharREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	chars, err := cr.m.getThingCharacteristics(id)
	if err != nil {
		return nil, err
	}
	return &runtime.ResponseModel{Characteristics: chars}, nil
}

type PSREST struct {
	m *Manager
}

func (ps *PSREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	pss, err := ps.m.getThingPropertySets(id)
	if err != nil {
		return nil, err
	}
	return &runtime.ResponseModel{PropertySets: pss}, nil
}

type TypeREST struct {
	m *Manager
}

var _ rest.Interface = (*TypeREST)(nil)
var _ rest.UpdateStrategy = (*TypeREST)(nil)

func (tr *TypeREST) New() gruntime.Object {
	return &v1.ThingType{}
}

func (tr *TypeREST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return tr.m.CreateThingType(obj.(*v1.ThingType))
}

func (tr *TypeREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	exploded, _ := strconv.ParseBool(options.Query.Get("exploded"))
	return tr.m.GetThingTypeById(id, exploded)
}

func (tr *TypeREST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	exploded, _ := strconv.ParseBool(options.Query.Get("exploded"))
	filter := ThingTypeFilter{Tenant: security.GetTenant()}
	if err := mapstructure.Decode(options.Filter, &filter); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	tts, _ := tr.m.GetThingTypes(&filter, exploded)
	return &runtime.ResponseModel{ThingTypes: tts}, nil
}

func (tr *TypeREST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	tt, err := tr.m.GetThingTypeById(id, false)
	if err != nil {
		return nil, err
	}
	copied := tt.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return tr.m.updateThingType(id, options.Version, obj.(*v1.ThingType), copied.(*runtime.ThingType))
}

func (tr *TypeREST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return tr.m.deleteThingType(id, options.Version)
}

func (tr *TypeREST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateThingType(obj.(*v1.ThingType))
}

func (tr *TypeREST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidateThingTypeUpdate(obj.(*v1.ThingType), old.(*runtime.ThingType))
}
