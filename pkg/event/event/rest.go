package event

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"lightiot/pkg/event/runtime"
	v1 "lightiot/pkg/event/v1"
	"lightiot/pkg/event/validation"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"strconv"
)

type TypeREST struct {
	m *Manager
}

func (tr *TypeREST) New() gruntime.Object {
	return &v1.EventType{}
}

func (tr *TypeREST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return tr.m.CreateEventType(obj.(*v1.EventType))
}

func (tr *TypeREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	exploded, _ := strconv.ParseBool(options.Query.Get("exploded"))
	return tr.m.GetEventTypeById(id, exploded)
}

func (tr *TypeREST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	exploded, _ := strconv.ParseBool(options.Query.Get("exploded"))
	filter := TypeFilter{}
	if err := mapstructure.Decode(options.Filter, &filter); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	ets, _ := tr.m.ListEventTypes(&filter, exploded)
	return &runtime.ResponseModel{EventTypes: ets}, nil
}

func (tr *TypeREST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	et, err := tr.m.getEventTypeById(id, true)
	if err != nil {
		return nil, err
	}
	copied := et.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return tr.m.UpdateEventType(id, options.Version, obj.(*v1.EventType), copied.(*runtime.EventType))
}

func (tr *TypeREST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return tr.m.DeleteEventType(id, options.Version)
}

func (r *TypeREST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateEventType(obj.(*v1.EventType))
}

func (r *TypeREST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidateEventTypeUpdate(obj.(*v1.EventType), old.(*runtime.EventType))
}
