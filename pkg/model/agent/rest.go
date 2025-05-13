package agent

import (
	"context"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"lightiot/pkg/model/validation"
	"os"
)

type REST struct {
	m *Manager
}

var _ rest.Interface = (*REST)(nil)
var _ rest.UpdateStrategy = (*REST)(nil)

func (r *REST) New() gruntime.Object {
	return &v1.Agent{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return r.m.CreateAgent(obj.(*v1.Agent))
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	t, isExist := r.m.agents.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return t.(gruntime.Object), nil
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	agents, _ := r.m.GetAgents(nil)
	return &runtime.ResponseModel{Agents: agents}, nil
}

func (r *REST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	a, err := r.m.GetAgentById(id)
	if err != nil {
		return nil, err
	}
	copied := a.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return r.m.updateAgent(id, options.Version, obj.(*v1.Agent), copied.(*runtime.Agent))
}

func (r *REST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return r.m.deleteAgent(id, options.Version)
}

func (r *REST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateAgent(obj.(*v1.Agent))
}

func (r *REST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidateAgentUpdate(obj.(*v1.Agent), old.(*runtime.Agent))
}

type TypeREST struct {
	m *Manager
}

var _ rest.Interface = (*TypeREST)(nil)
var _ rest.UpdateStrategy = (*TypeREST)(nil)

func (tr *TypeREST) New() gruntime.Object {
	return &v1.AgentType{}
}

func (tr *TypeREST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	return tr.m.CreateAgentType(obj.(*v1.AgentType))
}

func (tr *TypeREST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	t, isExist := tr.m.agentTypes.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return t.(gruntime.Object), nil
}

func (tr *TypeREST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	agentTypes, _ := tr.m.GetAgentTypes(nil)
	return &runtime.ResponseModel{AgentTypes: agentTypes}, nil
}

func (tr *TypeREST) Update(ctx context.Context, id string, obj gruntime.Object, updateValidation rest.ValidateObjectUpdateFunc, options *meta.UpdateOptions) (gruntime.Object, error) {
	at, err := tr.m.GetAgentTypeById(id)
	if err != nil {
		return nil, err
	}
	copied := at.DeepCopyObject()
	if err := updateValidation(ctx, obj, copied); err != nil {
		return nil, err
	}
	return tr.m.updateAgentType(id, options.Version, obj.(*v1.AgentType), copied.(*runtime.AgentType))
}

func (tr *TypeREST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return tr.m.deleteAgentType(id, options.Version)
}

func (tr *TypeREST) Validate(ctx context.Context, obj gruntime.Object) field.ErrorList {
	return validation.ValidateAgentType(obj.(*v1.AgentType))
}

func (tr *TypeREST) ValidateUpdate(ctx context.Context, obj, old gruntime.Object) field.ErrorList {
	return validation.ValidateAgentTypeUpdate(obj.(*v1.AgentType), old.(*runtime.AgentType))
}
