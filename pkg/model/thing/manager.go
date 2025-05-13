package thing

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/propertysettype"
	"lightiot/pkg/model/runtime"
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

type DeleteAgentByThingIdFunc func(thingId string) error
type DeleteMappingByThingPSNameFunc func(thingId, psName string)
type DeleteCommandTypeByThingTypeFunc func(thingTypeId string)

type Manager struct {
	thingTypes *sync.Map // map[id]*runtime.ThingType
	ttCh       chan gruntime.Object
	ttResCh    chan *storage.PersistResult
	ttStore    *generic.Store

	things *sync.Map // map[id]*runtime.Thing
	tCh    chan gruntime.Object
	tResCh chan *storage.PersistResult
	tStore *generic.Store

	stopCh <-chan struct{}

	pstm                         *propertysettype.Manager
	jm                           *job.Manager
	deleteAgentFunc              DeleteAgentByThingIdFunc
	deleteMappingFunc            DeleteMappingByThingPSNameFunc
	deleteCommandTypeByThingType DeleteCommandTypeByThingTypeFunc
	updateCacheFunc              propertysettype.UpdateCachedPropertySetTypeFunc

	Thing          *REST
	Characteristic *CharREST
	PropertySet    *PSREST
	ThingType      *TypeREST
}

type Option func(*Manager)

func WithThingTypeChan(ttCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.ttCh = ttCh
	}
}

func WithThingChan(tCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.tCh = tCh
	}
}

func WithPropertySetTypeManager(pstm *propertysettype.Manager) Option {
	return func(m *Manager) {
		m.pstm = pstm
	}
}

func WithJobManager(jm *job.Manager) Option {
	return func(m *Manager) {
		m.jm = jm
	}
}

func WithDeleteAgentFunc(f DeleteAgentByThingIdFunc) Option {
	return func(m *Manager) {
		m.deleteAgentFunc = f
	}
}

func WithDeleteMappingFunc(f DeleteMappingByThingPSNameFunc) Option {
	return func(m *Manager) {
		m.deleteMappingFunc = f
	}
}

func WithDeleteThingTypeFunc(f DeleteCommandTypeByThingTypeFunc) Option {
	return func(m *Manager) {
		m.deleteCommandTypeByThingType = f
	}
}

func WithUpdateCacheFunc(f propertysettype.UpdateCachedPropertySetTypeFunc) Option {
	return func(m *Manager) {
		m.updateCacheFunc = f
	}
}

func NewManager(stopCh <-chan struct{}, opts ...Option) *Manager {
	m := &Manager{
		thingTypes: &sync.Map{},
		things:     &sync.Map{},
		stopCh:     stopCh,
	}
	for _, opt := range opts {
		opt(m)
	}

	m.ttStore, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.ThingTypes}, &runtime.ThingType{}, m.ttCh)
	m.tStore, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.Things}, &runtime.Thing{}, m.tCh)

	m.Thing = &REST{m}
	m.Characteristic = &CharREST{m}
	m.PropertySet = &PSREST{m}
	m.ThingType = &TypeREST{m}
	return m
}

func (m *Manager) InitWrite() {
	m.load()
	m.ttResCh = m.ttStore.Start(m.stopCh)
	m.tResCh = m.tStore.Start(m.stopCh)
}

func (m *Manager) InitWatch() {
	m.load()
	_, _ = m.ttStore.Watch(m.stopCh, "", "")
	_, _ = m.tStore.Watch(m.stopCh, "", "")
	go m.processEvent(m.stopCh)
}

func (m *Manager) load() {
	instantiable := false
	description := "All agents' ancestor. If you need create an agent, you have to inherit it."
	baseAgent := &runtime.ThingType{
		ObjectMeta: meta.ObjectMeta{
			ID:      "main.BaseAgent",
			Tenant:  security.GetTenant(),
			Name:    "BaseAgent",
			Version: "0",
			ModTime: time.Time{},
		},
		Description:  &description,
		Instantiable: &instantiable,
	}
	m.thingTypes.Store(baseAgent.ID, baseAgent)

	tts, _ := m.ttStore.LoadResource()
	for _, obj := range tts {
		tt, _ := obj.(*runtime.ThingType)
		m.thingTypes.Store(tt.ID, tt)
	}
	if len(tts) > 0 {
		m.reindexThingTypes()
	}

	ts, _ := m.tStore.LoadResource()
	for _, obj := range ts {
		t, _ := obj.(*runtime.Thing)
		m.things.Store(t.ID, t)
	}
	if len(ts) > 0 {
		m.reindexThings()
	}
}

func (m *Manager) CreateThingType(obj *v1.ThingType) (*runtime.ThingType, error) {
	id := fmt.Sprintf("%s.%s", security.GetTenant(), obj.Name)
	if _, err := m.GetThingTypeById(id, false); err == nil {
		return nil, response.ErrResourceExists(obj.Name)
	}

	if len(obj.ParentTypeId) > 0 {
		if _, err := m.GetThingTypeById(obj.ParentTypeId, false); err != nil {
			return nil, response.ErrParentTypeNotFound(obj.ParentTypeId)
		}
	}

	tt := &runtime.ThingType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.TrimSpace(obj.Name),
			ID:      id,
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Description:          obj.Description,
		ParentTypeId:         obj.ParentTypeId,
		Instantiable:         obj.Instantiable,
		Characteristics:      []*runtime.Characteristic{},
		PropertySets:         []*runtime.PropertySet{},
		CharacteristicByName: make(map[string]*runtime.Characteristic),
		PropertySetByName:    make(map[string]*runtime.PropertySet),
	}

	SetDefaults_ThingType(tt)

	for _, c := range obj.Characteristics {
		cName := strings.TrimSpace(c.Name)
		if _, ok := tt.CharacteristicByName[cName]; ok {
			continue
		}

		rc := &runtime.Characteristic{
			Name:         cName,
			Unit:         c.Unit,
			Length:       c.Length,
			DataType:     c.DataType,
			DefaultValue: c.DefaultValue,
			Searchable:   c.Searchable,
		}

		tt.Characteristics = append(tt.Characteristics, rc)
		tt.CharacteristicByName[rc.Name] = rc
	}

	for _, ps := range obj.PropertySets {
		psName := strings.TrimSpace(ps.Name)
		if _, ok := tt.PropertySetByName[psName]; ok {
			continue
		}
		rps, err := m.createPropertySet(ps, tt)
		if err != nil {
			return nil, err
		}
		tt.PropertySets = append(tt.PropertySets, rps)
		tt.PropertySetByName[rps.Name] = rps
	}

	m.ttCh <- tt
	res := <-m.ttResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.thingTypes.Store(tt.ID, res.Saved)

	klog.V(2).InfoS("Created thingType", "id", tt.ID)
	return tt, nil
}

func (m *Manager) GetThingTypes(filter *ThingTypeFilter, exploded bool) ([]*runtime.ThingType, error) {
	tts := make([]*runtime.ThingType, 0)
	filter.Tenant = security.GetTenant()
	predicates := parseTypeFilter(filter)

	// descend
	byModTime := func(tt1, tt2 *runtime.ThingType) bool { return tt1.ModTime.Before(tt2.ModTime) }
	sorter := ByType(byModTime)

	m.thingTypes.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.ThingType)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			tts = sorter.Insert(tts, v)
		}
		return true
	})

	if exploded {
		for i := range tts {
			tts[i], _ = m.explodeThingType(tts[i])
		}
	}

	return tts, nil
}

func (m *Manager) getThingTypeByName(name string) *runtime.ThingType {
	var tt *runtime.ThingType = nil

	m.thingTypes.Range(func(key, value interface{}) bool {
		v, _ := value.(*runtime.ThingType)
		if name == v.Name {
			tt = v
			return false
		}
		return true
	})

	return tt
}

func (m *Manager) GetThingTypeById(id string, exploded bool) (*runtime.ThingType, error) {
	tt, isExist := m.thingTypes.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	rtt, _ := tt.(*runtime.ThingType)
	if exploded {
		rtt, _ = m.explodeThingType(rtt)
	}
	return rtt, nil
}

func (m *Manager) GetRootThingType(id string) (root *runtime.ThingType) {
	tt, _ := m.thingTypes.Load(id)
	if tt == nil {
		return
	}
	root = tt.(*runtime.ThingType)
	for len(root.ParentTypeId) != 0 {
		if tt, _ = m.thingTypes.Load(root.ParentTypeId); tt != nil {
			root = tt.(*runtime.ThingType)
		} else {
			return
		}
	}
	return
}

func (m *Manager) updateThingType(id, version string, obj *v1.ThingType, old *runtime.ThingType) (*runtime.ThingType, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}
	old.ModTime = time.Now()
	old.Description = obj.Description
	old.Instantiable = obj.Instantiable

	delChars, upChars, inChars := generic.DifferenceAndIntersectionObjects(old.Characteristics, obj.Characteristics,
		func(value interface{}) string { return value.(*runtime.Characteristic).Name },
		func(value interface{}) string { return value.(*v1.Characteristic).Name })

	// delete
	i := 0
	delCharSet := sets.NewString(delChars...)
	for _, c := range old.Characteristics {
		if !delCharSet.Has(c.Name) {
			old.Characteristics[i] = c
			i++
		} else {
			delete(old.CharacteristicByName, c.Name)
		}
	}
	for j := i; j < len(old.Characteristics); j++ {
		old.Characteristics[j] = nil
	}
	old.Characteristics = old.Characteristics[:i]

	// upsert
	for _, nc := range obj.Characteristics {
		name := strings.TrimSpace(nc.Name)
		if oc, ok := old.CharacteristicByName[name]; ok {
			oc.Unit = nc.Unit
			oc.Length = nc.Length
			oc.DefaultValue = nc.DefaultValue
		} else {
			rc := &runtime.Characteristic{
				Name:         name,
				Unit:         nc.Unit,
				Length:       nc.Length,
				DataType:     nc.DataType,
				DefaultValue: nc.DefaultValue,
				Searchable:   nc.Searchable,
			}
			old.Characteristics = append(old.Characteristics, rc)
			old.CharacteristicByName[name] = rc
		}
	}

	SetDefaults_ThingType(old)

	delPss, upPss, inPss := generic.DifferenceAndIntersectionObjects(old.PropertySets, obj.PropertySets,
		func(value interface{}) string { return value.(*runtime.PropertySet).Name },
		func(value interface{}) string { return value.(*v1.PropertySet).Name })

	// delete
	k := 0
	delPsSet := sets.NewString(delPss...)
	for _, ps := range old.PropertySets {
		if !delPsSet.Has(ps.Name) {
			old.PropertySets[k] = ps
			k++
		} else {
			delete(old.PropertySetByName, ps.Name)
		}
	}
	for j := k; j < len(old.PropertySets); j++ {
		old.PropertySets[j] = nil
	}
	old.PropertySets = old.PropertySets[:k]

	// upsert
	for _, nps := range obj.PropertySets {
		if ops, ok := old.PropertySetByName[nps.Name]; ok {
			// update
			if nps.PropertySetTypeId != nil && len(*nps.PropertySetTypeId) != 0 {
				if *nps.PropertySetTypeId != ops.PropertySetType.ID {
					// update shared pst
					pst, err := m.pstm.GetPropertySetTypeById(*nps.PropertySetTypeId)
					if err != nil {
						return nil, response.ErrPropertySetTypeNotFound(*nps.PropertySetTypeId)
					}
					ops.PropertySetType = pst
				}
			} else {
				// update exclusive pst
				ops.PropertySetType.ModTime = old.ModTime
				ops.PropertySetType.Description = nps.PropertySetType.Description
				for _, np := range nps.PropertySetType.Properties {
					name := strings.TrimSpace(np.Name)
					if op, ok := ops.PropertySetType.PropertyByName[name]; ok {
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
						ops.PropertySetType.Properties = append(ops.PropertySetType.Properties, rp)
						ops.PropertySetType.PropertyByName[name] = rp
					}
				}
			}
		} else {
			// insert
			rps, err := m.createPropertySet(nps, old)
			if err != nil {
				return nil, err
			}
			old.PropertySets = append(old.PropertySets, rps)
			old.PropertySetByName[rps.Name] = rps
		}
	}

	m.ttCh <- old
	res := <-m.ttResCh
	if res.Err != nil {
		return nil, res.Err
	}

	saved := res.Saved.(*runtime.ThingType)
	des, _ := m.thingTypes.Load(id)
	tt := des.(*runtime.ThingType)

	m.updateCachedThingType(saved, tt, delChars, upChars, inChars, delPss, upPss, inPss)

	klog.V(2).InfoS("Updated thingType", "id", tt.ID)
	return tt, nil
}

func (m *Manager) updateCachedThingType(new, old *runtime.ThingType, delChars, upChars, inChars, delPss, upPss, inPss []string) {
	// old.Version = new.Version
	old.ModTime = new.ModTime
	old.Description = new.Description
	old.Instantiable = new.Instantiable

	i := 0
	delCharSet := sets.NewString(delChars...)
	for _, c := range old.Characteristics {
		if !delCharSet.Has(c.Name) {
			old.Characteristics[i] = c
			i++
		} else {
			delete(old.CharacteristicByName, c.Name)
		}
	}
	for j := i; j < len(old.Characteristics); j++ {
		old.Characteristics[j] = nil
	}
	old.Characteristics = old.Characteristics[:i]

	for _, sc := range new.Characteristics {
		if c, ok := old.CharacteristicByName[sc.Name]; ok {
			*c = *sc
		} else {
			old.Characteristics = append(old.Characteristics, sc)
			old.CharacteristicByName[sc.Name] = sc
		}
	}

	k := 0
	delPsSet := sets.NewString(delPss...)
	for _, ps := range old.PropertySets {
		if !delPsSet.Has(ps.Name) {
			old.PropertySets[k] = ps
			k++
		} else {
			delete(old.PropertySetByName, ps.Name)
		}
	}
	for j := k; j < len(old.PropertySets); j++ {
		old.PropertySets[j] = nil
	}
	old.PropertySets = old.PropertySets[:k]

	for _, sps := range new.PropertySets {
		if ps, ok := old.PropertySetByName[sps.Name]; ok {
			*ps = *sps
		} else {
			old.PropertySets = append(old.PropertySets, sps)
			old.PropertySetByName[sps.Name] = sps
		}
	}

	m.updateDerivedChainThings(old, old, delChars, upChars, inChars, delPss, upPss, inPss)

	// keep at the bottom
	old.Version = new.Version
}

func (m *Manager) deleteThingType(id, version string) (*runtime.ThingType, error) {
	if id == "main.BaseAgent" {
		return nil, response.ErrDeleteFundamentalResourceFailed(id)
	}
	tt, err := m.GetThingTypeById(id, false)
	if err != nil {
		return nil, err
	}
	if m.hasChildThingType(id) {
		return nil, response.ErrChildTypeExist(id)
	}
	if tt.Version != version {
		return nil, apis.ErrMismatch
	}

	m.ttCh <- &runtime.ThingType{ObjectMeta: meta.ObjectMeta{ID: id, Version: version}}
	res := <-m.ttResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.thingTypes.Delete(id)

	_ = m.deleteThingsByThingTypeId(id)

	klog.V(2).InfoS("Deleted thingType", "id", tt.ID)
	return tt, nil
}

func (m *Manager) hasChildThingType(parentThingTypeId string) (has bool) {
	m.thingTypes.Range(func(key, value interface{}) bool {
		tt := value.(*runtime.ThingType)
		if tt.ParentTypeId == parentThingTypeId {
			has = true
			return false
		}
		return true
	})
	return
}

func (m *Manager) deleteThingsByThingTypeId(thingTypeId string) error {
	things, _ := m.GetThings(&thingFilter{TypeId: thingTypeId})
	for _, t := range things {
		func(t *runtime.Thing) {
			stop := make(chan struct{})
			go func() {
				wait.Until(func() {
					_, err := m.deleteThing(t.ID, t.Version)
					if err == nil {
						close(stop)
					}
				}, 0, stop)
			}()
		}(t)
	}
	return nil
}

func (m *Manager) explodeThingType(tt *runtime.ThingType) (*runtime.ThingType, error) {
	if len(tt.ParentTypeId) > 0 {
		rtt := &runtime.ThingType{
			ObjectMeta: meta.ObjectMeta{
				Tenant:  tt.Tenant,
				Name:    tt.Name,
				ID:      tt.ID,
				Version: tt.Version,
			},
			Description:          tt.Description,
			ParentTypeId:         tt.ParentTypeId,
			Instantiable:         tt.Instantiable,
			Characteristics:      []*runtime.Characteristic{},
			PropertySets:         []*runtime.PropertySet{},
			CharacteristicByName: make(map[string]*runtime.Characteristic),
			PropertySetByName:    make(map[string]*runtime.PropertySet),
		}

		id := tt.ID
		for len(id) > 0 {
			ptt, err := m.GetThingTypeById(id, false)
			if err != nil {
				klog.V(4).InfoS("ThingType not found", "id", id)
				break
			}
			for _, c := range ptt.Characteristics {
				if _, ok := rtt.CharacteristicByName[c.Name]; !ok {
					rtt.Characteristics = append(rtt.Characteristics, c)
					rtt.CharacteristicByName[c.Name] = c
				}
			}
			for _, ps := range ptt.PropertySets {
				if _, ok := rtt.PropertySetByName[ps.Name]; !ok {
					rtt.PropertySets = append(rtt.PropertySets, ps)
					rtt.PropertySetByName[ps.Name] = ps
				}
			}
			id = ptt.ParentTypeId
		}
		return rtt, nil
	}
	return tt, nil
}

func (m *Manager) createPropertySet(ps *v1.PropertySet, tt *runtime.ThingType) (*runtime.PropertySet, error) {
	rps := &runtime.PropertySet{
		Name: strings.TrimSpace(ps.Name),
	}

	if ps.PropertySetTypeId != nil && len(*ps.PropertySetTypeId) != 0 {
		// it's from an existed PropertySetType
		pst, err := m.pstm.GetPropertySetTypeById(*ps.PropertySetTypeId)
		if err != nil {
			return nil, response.ErrPropertySetTypeNotFound(*ps.PropertySetTypeId)
		}
		rps.PropertySetType = pst
	} else {
		// need create a new PropertySetType which is ONLY used by this ThingType
		rpst := &runtime.PropertySetType{
			ObjectMeta: meta.ObjectMeta{
				Tenant:  tt.Tenant,
				Name:    rps.Name,
				ID:      uuidutil.UUID(),
				ModTime: tt.ModTime,
			},
			Description:    ps.PropertySetType.Description,
			Properties:     []*runtime.Property{},
			PropertyByName: map[string]*runtime.Property{},
		}

		for _, p := range ps.PropertySetType.Properties {
			pName := strings.TrimSpace(p.Name)
			if _, ok := rpst.PropertyByName[pName]; ok {
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
			rpst.Properties = append(rpst.Properties, rp)
			rpst.PropertyByName[rp.Name] = rp
		}
		rps.PropertySetType = rpst
	}
	return rps, nil
}

type updateThingType struct {
	root, parent, child        *runtime.ThingType
	delChars, upChars, inChars []string
	delPss, upPss, inPss       []string
}

func (m *Manager) updateDerivedChainThings(root, tt *runtime.ThingType, delChars, upChars, inChars, delPss, upPss, inPss []string) {
	m.updateInstantiatedThings(root, tt, delChars, upChars, inChars, delPss, upPss, inPss)
	var utts []*updateThingType
	derivedTypes, _ := m.GetThingTypes(&ThingTypeFilter{ParentTypeId: tt.ID}, false)
	for _, dtt := range derivedTypes {
		utts = append(utts, &updateThingType{
			root:     root,
			parent:   tt,
			child:    dtt,
			delChars: delChars,
			upChars:  upChars,
			inChars:  inChars,
			delPss:   delPss,
			upPss:    upPss,
			inPss:    inPss,
		})
	}
	for len(utts) > 0 {
		var children []*updateThingType
		for _, utt := range utts {
			// cease the field update propagation that the derived field overrides super type
			_, intersectionChars, _ := generic.DifferenceAndIntersectionSameTypeObjects(utt.child.Characteristics, utt.parent.Characteristics, func(value interface{}) string {
				return value.(*runtime.Characteristic).Name
			})
			delChars, _, _ := generic.DifferenceAndIntersectionStrings(utt.delChars, intersectionChars)
			inChars, _, _ := generic.DifferenceAndIntersectionStrings(utt.inChars, intersectionChars)
			_, intersectionPss, _ := generic.DifferenceAndIntersectionSameTypeObjects(utt.child.PropertySets, utt.parent.PropertySets, func(value interface{}) string {
				return value.(*runtime.PropertySet).Name
			})
			delPss, _, _ := generic.DifferenceAndIntersectionStrings(utt.delPss, intersectionPss)
			inPss, _, _ := generic.DifferenceAndIntersectionStrings(utt.inPss, intersectionPss)

			// update
			m.updateInstantiatedThings(root, utt.child, delChars, nil, inChars, delPss, nil, inPss)
			tts, _ := m.GetThingTypes(&ThingTypeFilter{ParentTypeId: utt.child.ID}, false)
			for _, tt := range tts {
				children = append(children, &updateThingType{
					root:     root,
					parent:   utt.child,
					child:    tt,
					delChars: delChars,
					inChars:  inChars,
					delPss:   delPss,
					inPss:    inPss,
				})
			}
		}
		utts = children
	}
}

func (m *Manager) updateInstantiatedThings(root, tt *runtime.ThingType, delChars, upChars, inChars, delPss, upPss, inPss []string) {
	things, _ := m.GetThings(&thingFilter{TypeId: tt.ID})
	for _, t := range things {
		for _, name := range delChars {
			delete(t.CharacteristicByName, name)
			// TODO update things which override the characteristic default value
		}
		for _, name := range inChars {
			t.CharacteristicByName[name] = root.CharacteristicByName[name]
		}
		for _, name := range delPss {
			delete(t.PropertySetByName, name)
			if m.deleteMappingFunc != nil {
				m.deleteMappingFunc(t.ID, name)
			}
			// TODO delete related time series and roll ups
		}
		for _, name := range inPss {
			t.PropertySetByName[name] = root.PropertySetByName[name]
		}
	}
}

func (m *Manager) IsReferredByThingType(pstID string) bool {
	var referred bool
	m.thingTypes.Range(func(key, value interface{}) bool {
		tt, _ := value.(*runtime.ThingType)
		for _, ps := range tt.PropertySets {
			if ps.PropertySetType.ID == pstID {
				referred = true
				break
			}
		}
		return !referred
	})
	return referred
}

// thing

func (m *Manager) CreateThing(obj *v1.Thing) (*runtime.Thing, error) {
	tt, _ := m.GetThingTypeById(obj.TypeId, false)
	if tt == nil {
		return nil, response.ErrThingTypeNotFound(obj.TypeId)
	}

	if !*tt.Instantiable {
		return nil, response.ErrThingTypeNotInstantiable
	}

	parent, err := m.GetThingById(obj.ParentId)
	if len(obj.ParentId) > 0 && err != nil {
		return nil, response.ErrParentTypeNotFound(obj.ParentId)
	}

	var tz *time.Location = nil
	if obj.TimeZone != nil {
		tz, err = time.LoadLocation(*obj.TimeZone)
		if err != nil {
			klog.V(3).InfoS("Failed to parse time zone", "err", err)
			return nil, response.ErrTimeZoneInvalid(*obj.TimeZone)
		}
	}

	t := &runtime.Thing{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.TrimSpace(obj.Name),
			ID:      uuidutil.UUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Description:          obj.Description,
		Type:                 tt,
		Parent:               parent,
		TimeZone:             (*runtime.TimeZone)(tz),
		Characteristics:      []runtime.Value{},
		CharacteristicByName: make(map[string]*runtime.Characteristic),
		PropertySetByName:    make(map[string]*runtime.PropertySet),
	}

	SetDefaults_Thing(t)

	m.inheritCharacteristicAndPropertySet(t)

	for _, c := range obj.Characteristics {
		if _, ok := t.CharacteristicByName[c.Name]; !ok {
			return nil, response.ErrCharacteristicNotFound(c.Name)
		}
		value := runtime.Value{
			Name:       c.Name,
			Value:      c.Value,
			Definition: t.CharacteristicByName[c.Name],
		}
		t.Characteristics = append(t.Characteristics, value)
	}

	m.tCh <- t
	res := <-m.tResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.things.Store(t.ID, res.Saved)

	klog.V(2).InfoS("Created thing", "id", t.ID)
	return t, nil
}

func (m *Manager) getThingByName(name string) *runtime.Thing {
	var t *runtime.Thing = nil
	m.things.Range(func(key, value interface{}) bool {
		v, _ := value.(*runtime.Thing)
		if name == v.Name {
			t = v
			return false
		}
		return true
	})
	return t
}

func (m *Manager) GetThings(filter *thingFilter) ([]*runtime.Thing, error) {
	ts := make([]*runtime.Thing, 0)
	filter.Tenant = security.GetTenant()
	predicates := parseFilter(filter, m)

	// descend
	byModTime := func(t1, t2 *runtime.Thing) bool { return t1.ModTime.Before(t2.ModTime) }
	sorter := By(byModTime)

	m.things.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.Thing)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			ts = sorter.Insert(ts, v)
		}
		return true
	})

	return ts, nil
}

func (m *Manager) GetThingById(id string) (*runtime.Thing, error) {
	t, isExist := m.things.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return t.(*runtime.Thing), nil
}

func (m *Manager) deleteThing(id, version string) (*runtime.Thing, error) {
	tv, exist := m.things.Load(id)
	if !exist {
		return nil, os.ErrNotExist
	}

	t := tv.(*runtime.Thing)
	if t.Version != version {
		return nil, apis.ErrMismatch
	}

	m.tCh <- &runtime.Thing{ObjectMeta: meta.ObjectMeta{ID: id, Version: version}}
	res := <-m.tResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}

	m.things.Delete(id)
	_ = m.jm.DeleteThingJob(id, time.Now().UTC())

	children, _ := m.GetThings(&thingFilter{ParentId: id})

	for _, child := range children {
		func(t *runtime.Thing) {
			stop := make(chan struct{})
			go func() {
				wait.Until(func() {
					t.Parent = nil
					m.tCh <- t
					res := <-m.tResCh
					if res.Err == nil {
						saved, _ := res.Saved.(*runtime.Thing)
						saved.ModTime = time.Now()
						m.things.Store(t.ID, saved)
						close(stop)
					}
				}, 0, stop)
			}()
		}(child)
	}

	klog.V(2).InfoS("Deleted thing", "id", t.ID)

	_ = m.deleteAgentFunc(id)

	return t, nil
}

func (m *Manager) updateThing(id, version string, obj *v1.Thing, old *runtime.Thing) (*runtime.Thing, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}
	old.ModTime = time.Now()
	old.Description = obj.Description

	if obj.TimeZone != nil {
		tz, err := time.LoadLocation(*obj.TimeZone)
		if err != nil {
			klog.V(3).InfoS("Failed to parse time zone", "err", err)
			return nil, response.ErrTimeZoneInvalid(*obj.TimeZone)
		}
		old.TimeZone = (*runtime.TimeZone)(tz)
	}

	for _, c := range obj.Characteristics {
		if _, ok := old.CharacteristicByName[c.Name]; !ok {
			return nil, response.ErrCharacteristicNotFound(c.Name)
		}
	}

	if old.Parent == nil || obj.ParentId != old.Parent.ID {
		if len(obj.ParentId) == 0 {
			old.Parent = nil
		} else {
			parent, err := m.GetThingById(obj.ParentId)
			if len(obj.ParentId) > 0 && err != nil {
				return nil, response.ErrParentTypeNotFound(obj.ParentId)
			}
			if m.isMyChildThing(old.ID, obj.ParentId) {
				return nil, response.ErrCycledThing(obj.ParentId)
			}
			old.Parent = parent
		}
	}

	delChars, _, _ := generic.DifferenceAndIntersectionObjects(old.Characteristics, obj.Characteristics,
		func(value interface{}) string { return value.(runtime.Value).Name },
		func(value interface{}) string { return value.(v1.Value).Name })

	// delete
	i := 0
	delCharSet := sets.NewString(delChars...)
	for _, c := range old.Characteristics {
		if !delCharSet.Has(c.Name) {
			old.Characteristics[i] = c
		}
	}
	old.Characteristics = old.Characteristics[:i]

	// upsert
	for _, nc := range obj.Characteristics {
		var existed bool
		for _, oc := range old.Characteristics {
			if nc.Name == oc.Name {
				nc.Value = oc.Value
				existed = true
			}
		}
		if !existed {
			old.Characteristics = append(old.Characteristics, runtime.Value{
				Name:       nc.Name,
				Value:      nc.Value,
				Definition: old.CharacteristicByName[nc.Name],
			})
		}
	}

	m.tCh <- old
	res := <-m.tResCh
	if res.Err != nil {
		return nil, res.Err
	}

	saved := res.Saved.(*runtime.Thing)
	des, _ := m.things.Load(id)
	t := des.(*runtime.Thing)

	m.updateCachedThing(saved, t)

	klog.V(2).InfoS("Updated thing", "id", t.ID)
	return t, nil
}

func (m *Manager) updateCachedThing(new, old *runtime.Thing) {
	old.Version = new.Version
	old.ModTime = new.ModTime
	old.Description = new.Description
	old.TimeZone = new.TimeZone
	old.Parent = new.Parent
	old.Characteristics = new.Characteristics
}

func (m *Manager) getThingCharacteristics(id string) (interface{}, error) {
	obj, isExist := m.things.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}

	t := obj.(*runtime.Thing)

	type characteristic struct {
		Name       string      `json:"name"`
		Value      string      `json:"value"`
		Unit       string      `json:"unit"`
		Length     int         `json:"length"`
		DataType   v1.Datatype `json:"datatype"`
		Searchable bool        `json:"searchable"`
	}

	res := make([]*characteristic, 0, len(t.CharacteristicByName))

	overrideChars := sets.NewString()
	for _, c := range t.Characteristics {
		res = append(res, &characteristic{
			Name:       c.Name,
			Value:      c.Value,
			Unit:       c.Definition.Unit,
			Length:     c.Definition.Length,
			DataType:   c.Definition.DataType,
			Searchable: c.Definition.Searchable,
		})
		overrideChars.Insert(c.Name)
	}

	// inherit default value of characteristics
	for k, v := range t.CharacteristicByName {
		if !overrideChars.Has(k) {
			res = append(res, &characteristic{
				Name:       v.Name,
				Value:      v.DefaultValue,
				Unit:       v.Unit,
				Length:     v.Length,
				DataType:   v.DataType,
				Searchable: v.Searchable,
			})
		}
	}

	return res, nil
}

func (m *Manager) getThingPropertySets(id string) ([]*runtime.PropertySet, error) {
	t, isExist := m.things.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}

	pss := t.(*runtime.Thing).PropertySetByName
	res := make([]*runtime.PropertySet, len(pss))
	i := 0
	for _, v := range pss {
		res[i] = v
		i++
	}

	return res, nil
}

// if child has the same characteristic or property with parent,
// in other word they have the same name, the child shall hide
// its parent characteristic or propertySet. It is the same behavior
// with oriented object programming.
func (m *Manager) inheritCharacteristicAndPropertySet(t *runtime.Thing) {
	tcByName := t.CharacteristicByName
	tpsByName := t.PropertySetByName

	typeId := t.Type.ID
	for len(typeId) > 0 {
		tt, err := m.GetThingTypeById(typeId, false)
		if err != nil {
			klog.V(4).InfoS("ThingType not found", "id", typeId)
			return
		}
		for _, v := range tt.CharacteristicByName {
			if _, ok := tcByName[v.Name]; !ok {
				tcByName[v.Name] = v
			}
		}
		for _, v := range tt.PropertySets {
			if _, ok := tpsByName[v.Name]; !ok {
				tpsByName[v.Name] = v
			}
		}
		typeId = tt.ParentTypeId
	}
}

func (m *Manager) isMyChildThing(mySelf, id string) bool {
	children, _ := m.GetThings(&thingFilter{ParentId: mySelf})
	for len(children) > 0 {
		var ts []*runtime.Thing
		for _, child := range children {
			if child.ID == id {
				return true
			}
			ret, _ := m.GetThings(&thingFilter{ParentId: child.ID})
			ts = append(ts, ret...)
		}
		children = ts
	}
	return false
}

func (m *Manager) indexThingType(tt *runtime.ThingType) {
	tt.CharacteristicByName = make(map[string]*runtime.Characteristic, len(tt.Characteristics))
	for _, c := range tt.Characteristics {
		tt.CharacteristicByName[c.Name] = c
	}

	tt.PropertySetByName = make(map[string]*runtime.PropertySet, len(tt.PropertySets))
	for _, ps := range tt.PropertySets {
		if len(ps.PropertySetType.Name) == 0 {
			ps.PropertySetType, _ = m.pstm.GetPropertySetTypeById(ps.PropertySetType.ID)
		} else {
			ps.PropertySetType.PropertyByName = make(map[string]*runtime.Property, len(ps.PropertySetType.Properties))
			for _, p := range ps.PropertySetType.Properties {
				ps.PropertySetType.PropertyByName[p.Name] = p
			}
		}
		tt.PropertySetByName[ps.Name] = ps
	}
}

func (m *Manager) reindexThingTypes() {
	m.thingTypes.Range(func(key, value interface{}) bool {
		m.indexThingType(value.(*runtime.ThingType))
		return true
	})
}

func (m *Manager) indexThing(t *runtime.Thing) {
	tt, err := m.GetThingTypeById(t.Type.ID, false)
	if err != nil {
		klog.V(4).InfoS("ThingType not found", "id", t.Type.ID)
		return
	}
	t.Type = tt
	if t.Parent != nil && len(t.Parent.ID) > 0 {
		pt, err := m.GetThingById(t.Parent.ID)
		if err != nil {
			klog.V(4).InfoS("Thing's parent not found", "thingId", t.ID, "parentId", t.Parent.ID)
		}
		t.Parent = pt
	}

	t.CharacteristicByName = make(map[string]*runtime.Characteristic)
	t.PropertySetByName = make(map[string]*runtime.PropertySet)
	m.inheritCharacteristicAndPropertySet(t)
	for i := range t.Characteristics {
		t.Characteristics[i].Definition = t.CharacteristicByName[t.Characteristics[i].Name]
	}
}

func (m *Manager) reindexThings() {
	m.things.Range(func(key, value interface{}) bool {
		m.indexThing(value.(*runtime.Thing))
		return true
	})
}

func (m *Manager) processEvent(stopCh <-chan struct{}) {
	for {
		select {
		case obj, _ := <-m.ttCh:
			tt, _ := obj.(*runtime.ThingType)
			if tt.ModTime.IsZero() {
				klog.V(3).InfoS("Deleted thingType", "id", tt.ID)
				if old, ok := m.thingTypes.LoadAndDelete(tt.ID); ok {
					for _, ps := range old.(*runtime.ThingType).PropertySets {
						if !ps.PropertySetType.IsShared() && m.updateCacheFunc != nil {
							m.updateCacheFunc(ps.PropertySetType, runtime.Remove)
						}
					}
				}
				if m.deleteCommandTypeByThingType != nil {
					m.deleteCommandTypeByThingType(tt.ID)
				}
			} else if origin, ok := m.thingTypes.Load(tt.ID); ok {
				klog.V(3).InfoS("Updated thingType", "id", tt.ID)
				for _, ps := range tt.PropertySets {
					if len(ps.PropertySetType.Name) == 0 {
						ps.PropertySetType, _ = m.pstm.GetPropertySetTypeById(ps.PropertySetType.ID)
					}
				}
				old, _ := origin.(*runtime.ThingType)
				delChars, upChars, inChars := generic.DifferenceAndIntersectionSameTypeObjects(old.Characteristics, tt.Characteristics,
					func(value interface{}) string { return value.(*runtime.Characteristic).Name })
				delPss, upPss, inPss := generic.DifferenceAndIntersectionSameTypeObjects(old.PropertySets, tt.PropertySets,
					func(value interface{}) string { return value.(*runtime.PropertySet).Name })
				for _, del := range delPss {
					if !tt.PropertySetByName[del].PropertySetType.IsShared() && m.updateCacheFunc != nil {
						m.updateCacheFunc(tt.PropertySetByName[del].PropertySetType, runtime.Remove)
					}
				}
				for _, insert := range inPss {
					for i, nps := range tt.PropertySets {
						if insert == nps.Name && nps.PropertySetType.IsShared() {
							tt.PropertySets[i].PropertySetType, _ = m.pstm.GetPropertySetTypeById(nps.PropertySetType.ID)
						}
					}
				}

				m.updateCachedThingType(tt, old, delChars, upChars, inChars, delPss, upPss, inPss)

				for _, update := range upPss {
					if !tt.PropertySetByName[update].PropertySetType.IsShared() && m.updateCacheFunc != nil {
						m.updateCacheFunc(tt.PropertySetByName[update].PropertySetType, runtime.Update)
					}
				}
			} else {
				klog.V(3).InfoS("Created thingType", "id", tt.ID)
				m.indexThingType(tt)
				m.thingTypes.Store(tt.ID, tt)
			}
		case obj, _ := <-m.tCh:
			t, _ := obj.(*runtime.Thing)
			if t.ModTime.IsZero() {
				klog.V(3).InfoS("Deleted thing", "id", t.ID)
				m.things.Delete(t.ID)
			} else if old, ok := m.things.Load(t.ID); ok {
				klog.V(3).InfoS("Updated thing", "id", t.ID)
				m.updateCachedThing(t, old.(*runtime.Thing))
			} else {
				klog.V(3).InfoS("Created thing", "id", t.ID)
				m.indexThing(t)
				m.things.Store(t.ID, t)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped thing watch event")
			return
		}
	}
}
