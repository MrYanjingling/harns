package config

import (
	"context"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/notification/validation"
)

type REST struct {
	m *Manager
}

var (
	_ rest.Creater        = (*REST)(nil)
	_ rest.UpdateStrategy = (*REST)(nil)
	_ rest.Getter         = (*REST)(nil)
	_ rest.Lister         = (*REST)(nil)
	_ rest.Updater        = (*REST)(nil)
)

func NewREST(mgr *Manager) *REST {
	return &REST{m: mgr}
}

func (r *REST) New() gruntime.Object {
	return &v1.Config{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return r.m.createConfigs(obj.(*v1.Config))
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	servers, _ := r.m.GetConfig()
	rcs := make([]*runtime.Config, 0)
	if servers != nil {
		rcs = append(rcs, servers)
	}
	return &runtime.ResponseModel{ServerConfigurations: rcs}, nil
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	return r.m.GetConfig()
}

func (r *REST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	t, err := r.m.GetConfig()
	if err != nil {
		return nil, err
	}
	copied := t.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return r.m.updateConfigs(id, options.Version, obj.(*v1.Config), copied.(*runtime.Config))
}

func (r *REST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateServerConfig(obj.(*v1.Config))
}

func (r *REST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	// TODO should not allow delete any config. Or, the behavior of rule is not correct
	return nil
}
