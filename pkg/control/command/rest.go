package command

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"lightiot/pkg/control/runtime"
	v1 "lightiot/pkg/control/v1"
	"lightiot/pkg/control/validation"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/util/security"
	"strconv"
)

type TypeREST struct {
	m *Manager
}

func (tr *TypeREST) New() gruntime.Object {
	return &v1.CommandType{}
}

func (tr *TypeREST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	thingTypeId := options.Query.Get("thingTypeId")
	return tr.m.createCommandType(thingTypeId, obj.(*v1.CommandType))
}

func (tr *TypeREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	thingTypeId := options.Query.Get("thingTypeId")
	exploded, _ := strconv.ParseBool(options.Query.Get("exploded"))
	return tr.m.getCommandTypeById(thingTypeId, id, exploded)
}

func (tr *TypeREST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	thingTypeId := options.Query.Get("thingTypeId")
	f := typeFilter{Tenant: security.GetTenant()}
	if err := mapstructure.Decode(options.Filter, &f); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	if len(thingTypeId) != 0 {
		f.ThingTypeId = thingTypeId
	}
	exploded, _ := strconv.ParseBool(options.Query.Get("exploded"))
	cts, _ := tr.m.listCommandTypes(f, exploded)
	return &runtime.ResponseModel{CommandTypes: cts}, nil
}

func (tr *TypeREST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	ct, err := tr.m.getCommandTypeById(options.Query.Get("thingTypeId"), id, false)
	if err != nil {
		return nil, err
	}
	old := ct.DeepCopyObject()
	if err := updateValidation(ctx, obj, old); err != nil {
		return nil, err
	}
	return tr.m.updateCommandType(id, options.Version, obj.(*v1.CommandType), old.(*runtime.CommandType))
}

func (tr *TypeREST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	thingTypeId := options.Query.Get("thingTypeId")
	return tr.m.deleteCommandType(thingTypeId, id, options.Version)
}

func (tr *TypeREST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateCommandType(obj.(*v1.CommandType))
}

func (tr *TypeREST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidateCommandTypeUpdate(obj.(*v1.CommandType), old.(*runtime.CommandType))
}
