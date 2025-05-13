package propertysettype

import (
	"fmt"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IsReferredByThingTypeFunc func(pstID string) bool
type UpdateCachedPropertySetTypeFunc func(pst *runtime.PropertySetType, et runtime.EventType)

type Manager struct {
	propertySetTypes *sync.Map // id --> *runtime.PropertySetType
	pstCh            chan gruntime.Object
	pstResCh         chan *storage.PersistResult
	store            *generic.Store
	PST              *REST

	stopCh <-chan struct{}

	forbidDeletionFunc IsReferredByThingTypeFunc
	updateCacheFunc    UpdateCachedPropertySetTypeFunc
}

type Option func(*Manager)

func WithForbidDeletionFunc(f IsReferredByThingTypeFunc) Option {
	return func(m *Manager) {
		m.forbidDeletionFunc = f
	}
}

func WithUpdateCacheFunc(f UpdateCachedPropertySetTypeFunc) Option {
	return func(m *Manager) {
		m.updateCacheFunc = f
	}
}

func NewManager(stopCh <-chan struct{}, pstCh chan gruntime.Object, opts ...Option) *Manager {
	s, _ := generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.PropertySetTypes}, &runtime.PropertySetType{}, pstCh)

	m := &Manager{
		propertySetTypes: &sync.Map{},
		pstCh:            pstCh,
		store:            s,
		stopCh:           stopCh,
	}
	for _, opt := range opts {
		opt(m)
	}

	m.PST = &REST{m}

	return m
}

func (m *Manager) InitWrite() {
	m.load()
	m.pstResCh = m.store.Start(m.stopCh)
}

func (m *Manager) InitWatch() {
	m.load()
	_, _ = m.store.Watch(m.stopCh, "", "")
	go m.processEvent(m.stopCh)
}

func (m *Manager) load() {
	psts, _ := m.store.LoadResource()
	for _, obj := range psts {
		pst, _ := obj.(*runtime.PropertySetType)
		m.propertySetTypes.Store(pst.ID, pst)
	}
	if len(psts) > 0 {
		m.reindexPropertySetTypes()
	}
}

func (m *Manager) CreatePropertySetType(obj *v1.PropertySetType) (*runtime.PropertySetType, error) {
	id := fmt.Sprintf("%s.%s", security.GetTenant(), obj.Name)
	if _, err := m.GetPropertySetTypeById(id); err == nil {
		return nil, response.ErrResourceExists(id)
	}

	pst := &runtime.PropertySetType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.TrimSpace(obj.Name),
			ID:      id,
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Description:    obj.Description,
		Properties:     []*runtime.Property{},
		PropertyByName: map[string]*runtime.Property{},
	}

	for _, p := range obj.Properties {
		pName := strings.TrimSpace(p.Name)
		if _, ok := pst.PropertyByName[pName]; ok {
			continue
		}

		rp := &runtime.Property{
			Name:       pName,
			Unit:       p.Unit,
			Length:     p.Length,
			DataType:   p.Datatype,
			AccessMode: p.AccessMode,
			Min:        p.Min,
			Max:        p.Max,
		}
		pst.Properties = append(pst.Properties, rp)
		pst.PropertyByName[rp.Name] = rp
	}

	m.pstCh <- pst
	res := <-m.pstResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.propertySetTypes.Store(pst.ID, res.Saved)

	klog.V(2).InfoS("Created propertySetType", "id", pst.ID)
	return pst, nil
}

func (m *Manager) GetPropertySetTypes(filter *filter) ([]*runtime.PropertySetType, error) {
	psts := make([]*runtime.PropertySetType, 0)
	predicates := parseFilter(filter)

	// descend
	byModTime := func(pst1, pst2 *runtime.PropertySetType) bool { return pst1.ModTime.Before(pst2.ModTime) }
	sorter := By(byModTime)

	m.propertySetTypes.Range(func(key, value interface{}) bool {
		isMatch := true
		for _, p := range predicates {
			v := value.(*runtime.PropertySetType)
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			v, _ := value.(*runtime.PropertySetType)
			psts = sorter.Insert(psts, v)
		}
		return true
	})

	return psts, nil
}

func (m *Manager) getPropertySetByName(name string) *runtime.PropertySetType {
	var pst *runtime.PropertySetType = nil

	m.propertySetTypes.Range(func(key, value interface{}) bool {
		v, _ := value.(*runtime.PropertySetType)
		if name == v.Name {
			pst = v
			return false
		}
		return true
	})

	return pst
}

func (m *Manager) GetPropertySetTypeById(id string) (*runtime.PropertySetType, error) {
	pst, isExist := m.propertySetTypes.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return pst.(*runtime.PropertySetType), nil
}

func (m *Manager) updatePropertySetType(id, version string, obj *v1.PropertySetType, old *runtime.PropertySetType) (*runtime.PropertySetType, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}
	old.ModTime = time.Now()
	old.Description = obj.Description
	for _, np := range obj.Properties {
		name := strings.TrimSpace(np.Name)
		if op, ok := old.PropertyByName[name]; ok {
			op.AccessMode = np.AccessMode
			op.Min = np.Min
			op.Max = np.Max
		} else {
			rp := &runtime.Property{
				Name:       name,
				Unit:       np.Unit,
				Length:     np.Length,
				DataType:   np.Datatype,
				AccessMode: np.AccessMode,
				Min:        np.Min,
				Max:        np.Max,
			}
			old.Properties = append(old.Properties, rp)
			old.PropertyByName[name] = rp
		}
	}

	m.pstCh <- old
	res := <-m.pstResCh
	if res.Err != nil {
		return nil, res.Err
	}

	saved := res.Saved.(*runtime.PropertySetType)
	des, _ := m.propertySetTypes.Load(saved.ID)
	pst := des.(*runtime.PropertySetType)
	*pst = *saved

	klog.V(2).InfoS("Updated propertySetType", "id", pst.ID)
	return pst, nil
}

func (m *Manager) deletePropertySetType(propertySetTypeId, ETag string) (*runtime.PropertySetType, error) {
	pstv, exist := m.propertySetTypes.Load(propertySetTypeId)
	if !exist {
		return nil, os.ErrNotExist
	}

	pst := pstv.(*runtime.PropertySetType)
	if pst.Version != ETag {
		return nil, apis.ErrMismatch
	}

	if m.forbidDeletionFunc(propertySetTypeId) {
		return nil, response.ErrAssociateResourceExist(propertySetTypeId)
	}

	m.pstCh <- &runtime.PropertySetType{ObjectMeta: meta.ObjectMeta{ID: propertySetTypeId, Version: ETag}}
	res := <-m.pstResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.propertySetTypes.Delete(propertySetTypeId)

	klog.V(2).InfoS("Deleted propertySetType", "id", pst.ID)
	return pst, nil
}

func (m *Manager) reindexPropertySetTypes() {
	m.propertySetTypes.Range(func(key, value interface{}) bool {
		pst, _ := value.(*runtime.PropertySetType)
		pst.PropertyByName = make(map[string]*runtime.Property, len(pst.Properties))
		for _, p := range pst.Properties {
			pst.PropertyByName[p.Name] = p
		}
		return true
	})
}

func (m *Manager) processEvent(stopCh <-chan struct{}) {
	for {
		select {
		case obj, _ := <-m.pstCh:
			pst, _ := obj.(*runtime.PropertySetType)
			if pst.ModTime.IsZero() {
				klog.V(3).InfoS("Deleted propertySetType", "id", pst.ID)
				m.propertySetTypes.Delete(pst.ID)
				if m.updateCacheFunc != nil {
					m.updateCacheFunc(pst, runtime.Remove)
				}
			} else if origin, ok := m.propertySetTypes.Load(pst.ID); ok {
				klog.V(3).InfoS("Updated propertySetType", "id", pst.ID)
				old := origin.(*runtime.PropertySetType)
				for _, p := range pst.Properties {
					if _, ok := old.PropertyByName[p.Name]; !ok {
						old.Properties = append(old.Properties, p)
						old.PropertyByName[p.Name] = p
					}
				}
				if m.updateCacheFunc != nil {
					m.updateCacheFunc(pst, runtime.Update)
				}
			} else {
				klog.V(3).InfoS("Created propertySetType", "id", pst.ID)
				pst.PropertyByName = make(map[string]*runtime.Property, len(pst.Properties))
				for _, p := range pst.Properties {
					pst.PropertyByName[p.Name] = p
				}
				m.propertySetTypes.Store(pst.ID, pst)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped propertySetType watch event")
			return
		}
	}
}
