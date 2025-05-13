package datapointmapping

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"os"
)

type REST struct {
	m *Manager
}

var (
	_ rest.Creater = (*REST)(nil)
	_ rest.Lister  = (*REST)(nil)
	_ rest.Getter  = (*REST)(nil)
)

func (r *REST) New() gruntime.Object {
	return &v1.DataPointMapping{}
}

func (r *REST) Create(ctx context.Context, obj gruntime.Object, createValidation rest.ValidateObjectFunc, options *meta.CreateOptions) (gruntime.Object, error) {
	return r.m.createDataPointMapping(obj.(*v1.DataPointMapping))
}

func (r *REST) Get(ctx context.Context, id string, options *meta.GetOptions) (gruntime.Object, error) {
	t, isExist := r.m.mappings.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return t.(gruntime.Object), nil
}

func (r *REST) List(ctx context.Context, options *meta.ListOptions) (gruntime.Object, error) {
	f := Filter{}
	if err := mapstructure.Decode(options.Filter, &f); err != nil {
		klog.V(3).InfoS("Failed to parse filter", "err", err)
	}
	dpms, _ := r.m.getDataPointMappings(f)
	return &runtime.ResponseModel{DataPointMappings: dpms}, nil
}

func (r *REST) Delete(ctx context.Context, id string, options *meta.DeleteOptions) (gruntime.Object, error) {
	return r.m.deleteDataPointMappingsById(id)
}
