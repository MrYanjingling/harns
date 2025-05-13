package datapointmapping

import (
	"errors"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/callback"
	"lightiot/pkg/exchange"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/agent"
	"lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	v1 "lightiot/pkg/model/v1"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"os"
	"sync"
	"time"
)

type Manager struct {
	mappings *sync.Map // map[id]*runtime.DataPointMapping
	mCh      chan gruntime.Object
	mResCh   chan *storage.PersistResult
	store    *generic.Store

	stopCh <-chan struct{}

	tm *thing.Manager
	am *agent.Manager
	em *exchange.Manager

	Mapping *REST
}

type Option func(*Manager)

func WithMappingChan(mCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.mCh = mCh
	}
}

func WithThingManager(tm *thing.Manager) Option {
	return func(m *Manager) {
		m.tm = tm
	}
}

func WithAgentManager(am *agent.Manager) Option {
	return func(m *Manager) {
		m.am = am
	}
}

func WithExchangeManager(em *exchange.Manager) Option {
	return func(m *Manager) {
		m.em = em
	}
}

func NewManager(stopCh <-chan struct{}, opts ...Option) *Manager {
	m := &Manager{
		mappings: &sync.Map{},
		stopCh:   stopCh,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.store, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.Mappings}, &runtime.DataPointMapping{}, m.mCh)
	m.Mapping = &REST{m}
	return m
}

func (m *Manager) InitWrite() {
	m.load()
	m.mResCh = m.store.Start(m.stopCh)
}

func (m *Manager) InitWatch(cb callback.DataPointMappingHandler) {
	m.load()
	_, _ = m.store.Watch(m.stopCh, "", "")
	go m.processEvent(m.stopCh, cb)
}

func (m *Manager) load() {
	dpms := m.LoadMappings()
	for _, dpm := range dpms {
		m.mappings.Store(dpm.ID, dpm)
	}
}

func (m *Manager) LoadMappings() []*runtime.DataPointMapping {
	objs, _ := m.store.LoadResource()
	dpms := make([]*runtime.DataPointMapping, len(objs))
	for i := range objs {
		dpms[i] = objs[i].(*runtime.DataPointMapping)
	}
	return dpms
}

func (m *Manager) createDataPointMapping(obj *v1.DataPointMapping) (*runtime.DataPointMapping, error) {
	var err error
	a, err := m.am.GetAgentById(obj.AgentId)
	if err != nil {
		klog.V(3).InfoS("Agent not found", "id", obj.AgentId)
		return nil, response.ErrAgentNotFound(obj.AgentId)
	}

	var ds *runtime.DataSource = nil
	for _, v := range a.DataSources {
		if *v.Id == obj.DataSourceId {
			ds = v
		}
	}
	if ds == nil {
		klog.V(3).InfoS("DataSource not found", "dataSourceId", obj.DataSourceId, "agentId", obj.AgentId)
		return nil, response.ErrDataSourceNotFound(obj.DataSourceId)
	}

	var dp *runtime.DataPoint = nil
	dp, ok := ds.DataPointById[obj.DataPointId]
	if !ok {
		klog.V(3).InfoS("DataPoint not found", "dataPointId", obj.DataPointId, "dataSourceId", obj.DataSourceId, "agentId", obj.AgentId)
		return nil, response.ErrDataPointNotFound(obj.DataPointId)
	}

	t, err := m.tm.GetThingById(obj.ThingId)
	if err != nil {
		klog.V(3).InfoS("Thing not found", "id", obj.ThingId)
		return nil, response.ErrThingNotFound(obj.ThingId)
	}

	var ps *runtime.PropertySet = nil
	for _, v := range t.PropertySetByName {
		if v.Name == obj.PropertySetName {
			ps = v
		}
	}
	if ps == nil {
		klog.V(3).InfoS("PropertySet not found", "propertySet", obj.PropertySetName, "thingId", obj.ThingId)
		return nil, response.ErrPropertySetNotFound(obj.PropertySetName)
	}

	var p *runtime.Property = nil
	for _, v := range ps.PropertySetType.Properties {
		if v.Name == obj.PropertyName {
			p = v
		}
	}
	if p == nil {
		klog.V(3).InfoS("Property not found", "property", obj.PropertyName, "propertySet", obj.PropertySetName, "thingId", obj.ThingId)
		return nil, response.ErrPropertyNotFound(obj.PropertyName)
	}

	if dp.DataType != p.DataType {
		klog.V(3).InfoS("Mismatched datatype", "dataPoint", dp.DataType, "property", p.DataType)
		return nil, response.ErrDatatypeMismatch
	}

	if dp.Unit != p.Unit {
		klog.V(3).InfoS("Mismatched unit", "dataPoint", dp.Unit, "property", p.Unit)
		return nil, response.ErrDataUnitMismatch
	}

	if dp.AccessMode < p.AccessMode {
		klog.V(3).InfoS("Mismatched accessMode", "dataPoint", dp.AccessMode, "property", p.AccessMode)
		return nil, response.ErrAccessModeMismatch
	}

	isExist := false
	m.mappings.Range(func(key, value interface{}) bool {
		v := value.(*runtime.DataPointMapping)
		if v.ThingId == obj.ThingId &&
			v.PropertySetName == obj.PropertySetName &&
			v.PropertyName == obj.PropertyName {
			isExist = true
			return false
		}
		return true
	})

	if isExist {
		klog.V(3).InfoS("Existed dataPointMapping", "property", obj.PropertyName, "propertySet", obj.PropertySetName, "thingId", obj.ThingId)
		return nil, response.ErrDataPointMappingExists
	}

	rdpm := &runtime.DataPointMapping{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			ID:      uuidutil.UUID(),
			ModTime: time.Now(),
		},
		AgentId:            obj.AgentId,
		DataSourceId:       obj.DataSourceId,
		DataPointId:        obj.DataPointId,
		DataPointUnit:      dp.Unit,
		DataPointType:      dp.DataType,
		ThingId:            obj.ThingId,
		ThingName:          t.Name,
		PropertySetName:    obj.PropertySetName,
		PropertyName:       obj.PropertyName,
		PropertyUnit:       p.Unit,
		PropertyType:       p.DataType,
		PropertyAccessMode: p.AccessMode,
	}

	m.mCh <- rdpm
	res := <-m.mResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.mappings.Store(rdpm.ID, res.Saved)

	klog.V(2).InfoS("Created dataPointMapping", "id", rdpm.ID)
	return rdpm, nil
}

func (m *Manager) getDataPointMappings(filter Filter) ([]*runtime.DataPointMapping, error) {
	dpms := make([]*runtime.DataPointMapping, 0)
	predicates := parseFilter(filter)

	m.mappings.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.DataPointMapping)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			dpms = append(dpms, v)
		}
		return true
	})

	return dpms, nil
}

func (m *Manager) getDataPointMappingsById(id string) (*runtime.DataPointMapping, error) {
	a, isExist := m.mappings.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return a.(*runtime.DataPointMapping), nil
}

func (m *Manager) deleteDataPointMappingsById(id string) (*runtime.DataPointMapping, error) {
	dpm, isExist := m.mappings.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}

	m.mCh <- &runtime.DataPointMapping{ObjectMeta: meta.ObjectMeta{ID: id}}
	res := <-m.mResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.mappings.Delete(id)

	klog.V(2).InfoS("Deleted dataPointMapping", "id", id)

	return dpm.(*runtime.DataPointMapping), nil
}

func (m *Manager) DeleteMappingByAgentId(agentId string) error {
	dpms, _ := m.getDataPointMappings(Filter{AgentId: agentId})
	for _, dpm := range dpms {
		_, err := m.deleteDataPointMappingsById(dpm.ID)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (m *Manager) DeleteMappingByThingPSName(thingId, psName string) {
	dpms, _ := m.getDataPointMappings(Filter{ThingId: thingId, PropertySetName: psName})
	for _, dpm := range dpms {
		_, err := m.deleteDataPointMappingsById(dpm.ID)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			// should never be here
			klog.V(2).InfoS("Failed to delete", "id", dpm.ID, "err", err)
		}
	}
}

func (m *Manager) processEvent(stopCh <-chan struct{}, cb callback.DataPointMappingHandler) {
	for {
		select {
		case obj, _ := <-m.mCh:
			dpm, _ := obj.(*runtime.DataPointMapping)
			if dpm.ModTime.IsZero() {
				klog.V(3).InfoS("Deleted dataPointMapping", "id", dpm.ID)
				if d, ok := m.mappings.LoadAndDelete(dpm.ID); ok {
					cb(d.(*runtime.DataPointMapping), m.em, runtime.Remove)
				}
			} else {
				klog.V(3).InfoS("Created dataPointMapping", "id", dpm.ID)
				m.mappings.Store(dpm.ID, dpm)
				cb(dpm, m.em, runtime.Create)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped dataPointMapping watch event")
			return
		}
	}
}
