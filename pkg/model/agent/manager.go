package agent

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/callback"
	"lightiot/pkg/exchange"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	v1 "lightiot/pkg/model/v1"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DeleteMappingByAgentIdFunc func(agentId string) error

type Manager struct {
	agentTypes *sync.Map // id --> *runtime.AgentType
	atCh       chan gruntime.Object
	atResCh    chan *storage.PersistResult
	atStore    *generic.Store

	agents *sync.Map // id --> *runtime.Agent
	aCh    chan gruntime.Object
	aResCh chan *storage.PersistResult
	aStore *generic.Store

	stopCh <-chan struct{}

	tm *thing.Manager
	em *exchange.Manager

	deleteMappingFunc DeleteMappingByAgentIdFunc
	Agent             *REST
	AgentType         *TypeREST
}

type Option func(*Manager)

func WithAgentTypeChan(atCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.atCh = atCh
	}
}

func WithAgentChan(aCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.aCh = aCh
	}
}

func WithThingManager(tm *thing.Manager) Option {
	return func(m *Manager) {
		m.tm = tm
	}
}

func WithExchangeManager(em *exchange.Manager) Option {
	return func(m *Manager) {
		m.em = em
	}
}

func WithDeleteMappingByAgentIdFunc(f DeleteMappingByAgentIdFunc) Option {
	return func(m *Manager) {
		m.deleteMappingFunc = f
	}
}

func NewManager(stopCh <-chan struct{}, opts ...Option) *Manager {
	m := &Manager{
		agentTypes: &sync.Map{},
		agents:     &sync.Map{},
		stopCh:     stopCh,
	}
	for _, opt := range opts {
		opt(m)
	}

	m.atStore, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.AgentTypes}, &runtime.AgentType{}, m.atCh)
	m.aStore, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.Agents}, &runtime.Agent{}, m.aCh)

	m.Agent = &REST{m}
	m.AgentType = &TypeREST{m}
	return m
}

func (m *Manager) InitWrite() {
	m.load()
	m.atResCh = m.atStore.Start(m.stopCh)
	m.aResCh = m.aStore.Start(m.stopCh)
}

func (m *Manager) InitWatch(cb callback.AgentHandler) {
	m.load()
	_, _ = m.atStore.Watch(m.stopCh, "", "")
	_, _ = m.aStore.Watch(m.stopCh, "", "")
	go m.processEvent(m.stopCh, cb)
}

func (m *Manager) load() {
	ats, _ := m.atStore.LoadResource()
	for _, obj := range ats {
		at, _ := obj.(*runtime.AgentType)
		m.agentTypes.Store(at.ID, at)
	}
	if len(ats) > 0 {
		m.reindexAgentTypes()
	}

	as, _ := m.aStore.LoadResource()
	for _, obj := range as {
		a, _ := obj.(*runtime.Agent)
		m.agents.Store(a.ID, a)
	}
	if len(as) > 0 {
		m.reindexAgents()
	}
}

func (m *Manager) LoadAgents() []*runtime.Agent {
	objs, _ := m.aStore.LoadResource()
	as := make([]*runtime.Agent, len(objs))
	for i := range objs {
		as[i] = objs[i].(*runtime.Agent)
	}
	return as
}

func (m *Manager) CreateAgentType(obj *v1.AgentType) (*runtime.AgentType, error) {
	id := fmt.Sprintf("%s.%s", security.GetTenant(), obj.Name)
	if rat, err := m.GetAgentTypeById(id); err == nil {
		return rat, os.ErrExist
	}

	at := &runtime.AgentType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.TrimSpace(obj.Name),
			ID:      id,
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Description:      obj.Description,
		DataSources:      []*runtime.DataSource{},
		DataSourceByName: make(map[string]*runtime.DataSource),
	}
	createTypeDataSources(obj.DataSources, at)

	m.atCh <- at
	res := <-m.atResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.agentTypes.Store(at.ID, res.Saved)

	klog.V(2).InfoS("Created agentType", "id", at.ID)
	return at, nil
}

func (m *Manager) GetAgentTypes(filter map[string]interface{}) ([]*runtime.AgentType, error) {
	ats := make([]*runtime.AgentType, 0)
	predicates := make([]runtime.Predicate, 0)

	if len(filter) >= 0 {
		if name, ok := filter["name"]; ok {
			p := func(value interface{}) bool {
				v, _ := value.(*runtime.AgentType)
				if name == v.Name {
					return true
				}
				return false
			}
			predicates = append(predicates, p)
		}
	}

	// descend
	byModTime := func(at1, at2 *runtime.AgentType) bool { return at1.ModTime.Before(at2.ModTime) }
	sorter := ByType(byModTime)

	m.agentTypes.Range(func(key, value interface{}) bool {
		isMatch := true
		for _, p := range predicates {
			if !p(value) {
				isMatch = false
				break
			}
		}
		if isMatch {
			v, _ := value.(*runtime.AgentType)
			ats = sorter.Insert(ats, v)
		}
		return true
	})

	return ats, nil
}

func (m *Manager) getAgentTypeByName(name string) *runtime.AgentType {
	var at *runtime.AgentType = nil

	m.agentTypes.Range(func(key, value interface{}) bool {
		v, _ := value.(*runtime.AgentType)
		if name == v.Name {
			at = v
			return false
		}
		return true
	})

	return at
}

func (m *Manager) GetAgentTypeById(id string) (*runtime.AgentType, error) {
	at, isExist := m.agentTypes.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return at.(*runtime.AgentType), nil
}

func (m *Manager) updateAgentType(id, version string, obj *v1.AgentType, old *runtime.AgentType) (*runtime.AgentType, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}
	old.ModTime = time.Now()
	old.Description = obj.Description
	createTypeDataSources(obj.DataSources, old)

	m.atCh <- old
	res := <-m.atResCh
	if res.Err != nil {
		return nil, res.Err
	}

	saved := res.Saved.(*runtime.AgentType)
	des, _ := m.agentTypes.Load(id)
	at := des.(*runtime.AgentType)
	*at = *saved
	return at, nil
}

func (m *Manager) deleteAgentType(agentTypeId, version string) (*runtime.AgentType, error) {
	atv, exist := m.agentTypes.Load(agentTypeId)
	if !exist {
		return nil, os.ErrNotExist
	}

	at := atv.(*runtime.AgentType)
	if at.Version != version {
		return nil, apis.ErrMismatch
	}

	m.atCh <- &runtime.AgentType{ObjectMeta: meta.ObjectMeta{ID: agentTypeId, Version: version}}
	res := <-m.atResCh
	if res.Err != nil {
		return nil, res.Err
	}

	m.agentTypes.Delete(agentTypeId)

	klog.V(2).InfoS("Deleted agentType", "id", at.ID)
	return at, nil
}

func createTypeDataSources(dataSources []*v1.DataSource, at *runtime.AgentType) {
	for _, ds := range dataSources {
		dsName := strings.TrimSpace(ds.Name)
		if _, ok := at.DataSourceByName[dsName]; ok {
			continue
		}
		rds := &runtime.DataSource{
			Id:            nil,
			Name:          dsName,
			Description:   ds.Description,
			DataPoints:    []*runtime.DataPoint{},
			CustomData:    ds.CustomData,
			DataPointById: make(map[string]*runtime.DataPoint),
		}

		for _, dp := range ds.DataPoints {
			dpId := strings.TrimSpace(dp.Id)
			if _, ok := rds.DataPointById[dpId]; ok {
				continue
			}
			rdp := &runtime.DataPoint{
				Id:          dpId,
				Name:        strings.TrimSpace(dp.Name),
				Description: dp.Description,
				DataType:    dp.Datatype,
				Unit:        dp.Unit,
				AccessMode:  dp.AccessMode,
				CustomData:  dp.CustomData,
			}
			rds.DataPoints = append(rds.DataPoints, rdp)
			rds.DataPointById[rdp.Id] = rdp
		}

		at.DataSources = append(at.DataSources, rds)
		at.DataSourceByName[rds.Name] = rds
	}
}

// agent

func (m *Manager) CreateAgent(obj *v1.Agent) (*runtime.Agent, error) {
	var at *runtime.AgentType
	if obj.TypeId != nil && len(*obj.TypeId) > 0 {
		at, _ = m.GetAgentTypeById(*obj.TypeId)
		if at == nil {
			return nil, response.ErrAgentTypeNotFound(*obj.TypeId)
		}
	}

	if _, err := m.tm.GetThingById(obj.ThingId); err != nil {
		return nil, response.ErrThingNotFound(obj.ThingId)
	}

	a := &runtime.Agent{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.TrimSpace(obj.Name),
			ID:      obj.ThingId,
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		ThingId:          obj.ThingId,
		Description:      obj.Description,
		DataSources:      []*runtime.DataSource{},
		DataSourceByName: make(map[string]*runtime.DataSource),
	}

	if at != nil {
		a.TypeId = &at.ID
		m.inheritDataSourceAndDataPoint(at.DataSources, a)
	} else {
		for _, ds := range obj.DataSources {
			dsName := strings.TrimSpace(ds.Name)
			if _, ok := a.DataSourceByName[dsName]; ok {
				continue
			}
			id := uuidutil.UUID()
			rds := &runtime.DataSource{
				Id:            &id,
				Name:          dsName,
				Description:   ds.Description,
				DataPoints:    []*runtime.DataPoint{},
				CustomData:    ds.CustomData,
				DataPointById: make(map[string]*runtime.DataPoint),
			}

			for _, dp := range ds.DataPoints {
				dpId := strings.TrimSpace(dp.Id)
				if _, ok := rds.DataPointById[dpId]; ok {
					continue
				}
				rdp := &runtime.DataPoint{
					Id:          dpId,
					Name:        dp.Name,
					Description: dp.Description,
					DataType:    dp.Datatype,
					Unit:        dp.Unit,
					AccessMode:  dp.AccessMode,
					CustomData:  dp.CustomData,
				}
				rds.DataPoints = append(rds.DataPoints, rdp)
				rds.DataPointById[rdp.Id] = rdp
			}

			a.DataSources = append(a.DataSources, rds)
			a.DataSourceByName[rds.Name] = rds
		}
	}

	m.aCh <- a
	res := <-m.aResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.agents.Store(a.ID, res.Saved)

	klog.V(2).InfoS("Created agent", "id", a.ID)
	return a, nil
}

func (m *Manager) getAgentByName(name string) *runtime.Agent {
	var a *runtime.Agent = nil
	m.agents.Range(func(key, value interface{}) bool {
		v, _ := value.(*runtime.Agent)
		if name == v.Name {
			a = v
			return false
		}
		return true
	})
	return a
}

func (m *Manager) GetAgents(filter map[string]interface{}) ([]*runtime.Agent, error) {
	as := make([]*runtime.Agent, 0)
	predicates := make([]runtime.Predicate, 0)

	if len(filter) >= 0 {
		if name, ok := filter["name"]; ok {
			p := func(value interface{}) bool {
				v, _ := value.(*runtime.Agent)
				if name == v.Name {
					return true
				}
				return false
			}
			predicates = append(predicates, p)
		}
	}

	// descend
	byModTime := func(a1, a2 *runtime.Agent) bool { return a1.ModTime.Before(a2.ModTime) }
	sorter := By(byModTime)

	m.agents.Range(func(key, value interface{}) bool {
		isMatch := true
		for _, p := range predicates {
			if !p(value) {
				isMatch = false
				break
			}
		}
		if isMatch {
			v, _ := value.(*runtime.Agent)
			as = sorter.Insert(as, v)
		}
		return true
	})

	return as, nil
}

func (m *Manager) GetAgentById(id string) (*runtime.Agent, error) {
	a, isExist := m.agents.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return a.(*runtime.Agent), nil
}

func (m *Manager) updateAgent(agentId, version string, obj *v1.Agent, old *runtime.Agent) (*runtime.Agent, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}
	old.ModTime = time.Now()
	old.Description = obj.Description

	delDSs, _, _ := generic.DifferenceAndIntersectionObjects(old.DataSources, obj.DataSources,
		func(value interface{}) string { return value.(*runtime.DataSource).Name },
		func(value interface{}) string { return value.(*v1.DataSource).Name })

	// delete
	i := 0
	delDSSet := sets.NewString(delDSs...)
	for _, ds := range old.DataSources {
		if !delDSSet.Has(ds.Name) {
			old.DataSources[i] = ds
			i++
		} else {
			delete(old.DataSourceByName, ds.Name)
		}
	}
	for j := i; j < len(old.DataSources); j++ {
		old.DataSources[j] = nil
	}
	old.DataSources = old.DataSources[:i]

	// upsert
	for _, nds := range obj.DataSources {
		if ods, ok := old.DataSourceByName[nds.Name]; ok {
			updateDataSource(nds, ods)
		} else {
			rds := createDatasource(nds)
			old.DataSources = append(old.DataSources, rds)
			old.DataSourceByName[nds.Name] = rds
		}
	}

	m.aCh <- old
	res := <-m.aResCh
	if res.Err != nil {
		return nil, res.Err
	}

	saved := res.Saved.(*runtime.Agent)
	des, _ := m.agents.Load(agentId)
	a := des.(*runtime.Agent)
	*a = *saved

	_ = m.deleteMappingFunc(agentId)

	klog.V(2).InfoS("Updated agent", "id", a.ID)
	return a, nil
}

func (m *Manager) inheritDataSourceAndDataPoint(dataSources []*runtime.DataSource, a *runtime.Agent) {
	for _, ds := range dataSources {
		id := uuidutil.UUID()
		ads := &runtime.DataSource{
			Id:            &id,
			Name:          ds.Name,
			Description:   ds.Description,
			DataPoints:    []*runtime.DataPoint{},
			CustomData:    ds.CustomData,
			DataPointById: make(map[string]*runtime.DataPoint),
		}

		for _, v := range ds.DataPoints {
			dp := &runtime.DataPoint{
				Id:          v.Id,
				Name:        v.Name,
				Description: v.Description,
				DataType:    v.DataType,
				Unit:        v.Unit,
				AccessMode:  v.AccessMode,
				CustomData:  v.CustomData,
			}

			ads.DataPoints = append(ads.DataPoints, dp)
			ads.DataPointById[dp.Id] = dp
		}

		a.DataSources = append(a.DataSources, ads)
		a.DataSourceByName[ads.Name] = ads
	}
}

func (m *Manager) reindexAgentTypes() {
	m.agentTypes.Range(func(key, value interface{}) bool {
		at, _ := value.(*runtime.AgentType)
		at.DataSourceByName = make(map[string]*runtime.DataSource, len(at.DataSources))
		for _, ds := range at.DataSources {
			at.DataSourceByName[ds.Name] = ds
			ds.DataPointById = make(map[string]*runtime.DataPoint, len(ds.DataPoints))
			for _, dp := range ds.DataPoints {
				ds.DataPointById[dp.Id] = dp
			}
		}
		return true
	})
}

func (m *Manager) reindexAgents() {
	m.agents.Range(func(key, value interface{}) bool {
		a, _ := value.(*runtime.Agent)
		a.DataSourceByName = make(map[string]*runtime.DataSource)
		for _, ds := range a.DataSources {
			ds.DataPointById = make(map[string]*runtime.DataPoint)
			for _, dp := range ds.DataPoints {
				ds.DataPointById[dp.Id] = dp
			}
			a.DataSourceByName[ds.Name] = ds
		}
		return true
	})
}

func (m *Manager) deleteAgent(agentId, version string) (*runtime.Agent, error) {
	av, exist := m.agents.Load(agentId)
	if !exist {
		return nil, os.ErrNotExist
	}

	a := av.(*runtime.Agent)
	if a.Version != version {
		return nil, apis.ErrMismatch
	}

	m.aCh <- &runtime.Agent{ObjectMeta: meta.ObjectMeta{ID: agentId, Version: version}}
	res := <-m.aResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.agents.Delete(agentId)
	_ = m.deleteMappingFunc(agentId)

	klog.V(2).InfoS("Deleted agent", "id", a.ID)
	return a, nil
}

func (m *Manager) DeleteAgentByThingId(thingId string) error {
	m.aCh <- &runtime.Agent{ObjectMeta: meta.ObjectMeta{ID: thingId}}
	res := <-m.aResCh
	if res.Err != nil {
		return res.Err
	}
	m.agents.Delete(thingId)
	_ = m.deleteMappingFunc(thingId)
	return nil
}

func (m *Manager) processEvent(stopCh <-chan struct{}, cb callback.AgentHandler) {
	for {
		select {
		// case at, _ := <- m.atCh:
		// do not care the change of agentType
		case obj, _ := <-m.aCh:
			a, _ := obj.(*runtime.Agent)
			if a.ModTime.IsZero() {
				if av, exist := m.agents.LoadAndDelete(a.ID); exist {
					klog.V(3).InfoS("Deleted agent", "id", a.ID)
					cb(av.(*runtime.Agent), nil, m.em, runtime.Remove)
				}
			} else if origin, ok := m.agents.Load(a.ID); ok {
				klog.V(3).InfoS("Updated agent", "id", a.ID)
				cb(a, origin.(*runtime.Agent), m.em, runtime.Update)
			} else {
				// TODO used in get when delete. But we need not agent raw info
				klog.V(3).InfoS("Created agent", "id", a.ID)
				m.agents.Store(a.ID, a)
				cb(a, nil, m.em, runtime.Create)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped agent watch event")
			return
		}
	}
}

func updateDataSource(obj *v1.DataSource, old *runtime.DataSource) {
	old.Description = obj.Description
	old.CustomData = obj.CustomData
	dps := make([]*runtime.DataPoint, len(obj.DataPoints))
	dpByID := make(map[string]*runtime.DataPoint, len(obj.DataPoints))
	for i, dp := range obj.DataPoints {
		dpId := strings.TrimSpace(dp.Id)
		rdp := &runtime.DataPoint{
			Id:          dpId,
			Name:        dp.Name,
			Description: dp.Description,
			DataType:    dp.Datatype,
			Unit:        dp.Unit,
			AccessMode:  dp.AccessMode,
			CustomData:  dp.CustomData,
		}
		dps[i] = rdp
		dpByID[rdp.Id] = rdp
	}
	old.DataPoints = dps
	old.DataPointById = dpByID
}

func createDatasource(ds *v1.DataSource) *runtime.DataSource {
	id := uuidutil.UUID()
	rds := &runtime.DataSource{
		Id:            &id,
		Name:          ds.Name,
		Description:   ds.Description,
		DataPoints:    make([]*runtime.DataPoint, 0, len(ds.DataPoints)),
		CustomData:    ds.CustomData,
		DataPointById: make(map[string]*runtime.DataPoint, len(ds.DataPoints)),
	}

	for _, dp := range ds.DataPoints {
		rdp := &runtime.DataPoint{
			Id:          dp.Id,
			Name:        dp.Name,
			Description: dp.Description,
			DataType:    dp.Datatype,
			Unit:        dp.Unit,
			AccessMode:  dp.AccessMode,
			CustomData:  dp.CustomData,
		}
		rds.DataPoints = append(rds.DataPoints, rdp)
		rds.DataPointById[dp.Id] = rdp
	}
	return rds
}
