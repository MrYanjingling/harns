package messagetemplate

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/notification/validation"
	"lightiot/pkg/util/security"
	"strconv"
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
	return &v1.MessageTmpl{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return r.m.CreateMessageTmpls(obj.(*v1.MessageTmpl))
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	return r.m.GetMessageTmplById(id, false)
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	filter := Filter{Tenant: security.GetTenant()}
	if err := mapstructure.Decode(options.Filter, &filter); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	rmts, _ := r.m.ListMessageTmpls(filter)
	return &runtime.ResponseModel{MessageTemplates: rmts}, nil
}

func (r *REST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	// TODO
	return nil, nil
}

func (r *REST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	orphanRemoval, _ := strconv.ParseBool(options.Query.Get(apis.OrphanRemoval))
	return r.m.DeleteMsgTmpl(id, options.Version, orphanRemoval)
}

func (r *REST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateMessageTmpl(obj.(*v1.MessageTmpl))
}

func (r *REST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	// TODO
	return nil
}

type PlaceHolderREST struct {
	m *Manager
}

var (
	_ rest.Getter         = (*PlaceHolderREST)(nil)
	_ rest.Updater        = (*PlaceHolderREST)(nil)
	_ rest.UpdateStrategy = (*PlaceHolderREST)(nil)
)

func NewRESTPlaceHolder(mgr *Manager) *PlaceHolderREST {
	return &PlaceHolderREST{m: mgr}
}

func (phr *PlaceHolderREST) New() gruntime.Object {
	return &runtime.TemplatePlaceholder{}
}

func (phr *PlaceHolderREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	rph, err := phr.m.getPlaceholders(id)
	if err != nil {
		return nil, err
	}
	return &runtime.ResponseModel{Placeholders: rph}, nil
}

func (phr *PlaceHolderREST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	ph, err := phr.m.getPlaceholders(id)
	if err != nil {
		return nil, err
	}
	copied := ph.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return phr.m.updatePlaceholders(id, options.Version, obj.(*runtime.TemplatePlaceholder), ph)
}

func (phr *PlaceHolderREST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return nil
}

func (phr *PlaceHolderREST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return nil
}
