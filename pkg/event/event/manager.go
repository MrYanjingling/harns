package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/event/runtime"
	"lightiot/pkg/event/storage"
	v1 "lightiot/pkg/event/v1"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	model "lightiot/pkg/model/runtime"
	fstorage "lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	eventTypes *sync.Map // map[id]*runtime.EventType
	etCh       chan gruntime.Object
	etResCh    chan *fstorage.PersistResult

	things *sync.Map // map[ThingId]map[EventTypeId]struct{}{}
	tCh    chan gruntime.Object

	stopCh <-chan struct{}

	store     *storage.Store
	typeStore *storage.TypeStore

	jm       *job.Manager
	validate *validator.Validate

	EventType *TypeREST
}

func NewManager(stopCh <-chan struct{}, store *storage.Store, typeStore *storage.TypeStore, etCh chan gruntime.Object, jm *job.Manager, tCh chan gruntime.Object) *Manager {
	m := &Manager{
		eventTypes: &sync.Map{},
		etCh:       etCh,
		things:     &sync.Map{},
		tCh:        tCh,
		stopCh:     stopCh,
		store:      store,
		typeStore:  typeStore,
		jm:         jm,
		validate:   validator.New(),
	}

	m.EventType = &TypeREST{m}
	return m
}

func (m *Manager) Init() {
	baseFields := []*runtime.Field{
		{runtime.BaseEventFieldId, true, false, false, v1.DatatypeString, nil},
		{runtime.BaseEventFieldTypeId, true, false, false, v1.DatatypeString, nil},
		{runtime.BaseEventFieldCorrelationId, true, false, false, v1.DatatypeString, nil},
		{runtime.BaseEventFieldTime, false, true, false, v1.DatatypeString, nil},
		{runtime.BaseEventFieldThingId, true, true, false, v1.DatatypeString, nil},
		{runtime.BaseEventFieldETag, false, false, false, v1.DatatypeString, nil},
	}
	bet := &runtime.EventType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.Split(runtime.BaseEventTypeId, ".")[1],
			ID:      runtime.BaseEventTypeId,
			Version: "0",
			ModTime: time.Time{},
		},
		TTL:         30,
		Fields:      baseFields,
		FieldByName: make(map[string]*runtime.Field, len(baseFields)),
	}
	for _, f := range bet.Fields {
		bet.FieldByName[f.Name] = f
	}
	m.eventTypes.Store(bet.ID, bet)

	standardFields := []*runtime.Field{
		{runtime.StandardEventFieldSeverity, true, false, false, v1.DatatypeInt, nil},
		{runtime.StandardEventFieldDescription, false, false, false, v1.DatatypeString, nil},
		{runtime.StandardEventFieldCode, true, false, false, v1.DatatypeString, nil},
		{runtime.StandardEventFieldSource, true, false, false, v1.DatatypeString, nil},
		{runtime.StandardEventFieldAcknowledged, true, false, true, v1.DatatypeBool, nil},
	}
	standardFields = append(standardFields, baseFields...)
	set := &runtime.EventType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.Split(runtime.StandardEventTypeId, ".")[1],
			ID:      runtime.StandardEventTypeId,
			Version: "0",
			ModTime: time.Time{},
		},
		TTL:         7,
		Fields:      standardFields,
		FieldByName: make(map[string]*runtime.Field, len(standardFields)),
	}
	for _, f := range set.Fields {
		set.FieldByName[f.Name] = f
	}
	m.eventTypes.Store(set.ID, set)

	ets, _ := m.typeStore.LoadEventTypes()

	for _, et := range ets {
		m.eventTypes.Store(et.ID, et)
	}

	if len(ets) > 0 {
		m.reindexEventType()
	}

	modelThings, activeThings, eventTypesByThing, _ := m.typeStore.LoadThings()
	for _, t := range modelThings.UnsortedList() {
		var ets *sync.Map
		if activeThings.Has(t) {
			ets = &sync.Map{}
			if et, ok := eventTypesByThing[t]; ok {
				for _, k := range et.UnsortedList() {
					ets.Store(k, struct{}{})
				}
			}
		}
		m.things.Store(t, ets)
	}

	m.etResCh = m.typeStore.Start(m.stopCh)

	_ = m.typeStore.PruneThings(activeThings.Difference(modelThings))

	go m.processEvent(m.stopCh)
}

func (m *Manager) CreateEventType(obj *v1.EventType) (*runtime.EventType, error) {
	var id string
	if obj.Id == "" || !strings.HasPrefix(obj.Id, security.GetTenant()+".") {
		id = fmt.Sprintf("%s.%s", security.GetTenant(), obj.Name)
	} else {
		id = obj.Id
	}
	if _, err := m.getEventTypeById(id, true); err == nil {
		return nil, response.ErrResourceExists(id)
	}

	if obj.ParentId != nil && len(*obj.ParentId) > 0 {
		if _, err := m.getEventTypeById(*obj.ParentId, true); err != nil {
			return nil, response.ErrResourceNotFound(*obj.ParentId)
		}
	}

	et := &runtime.EventType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    obj.Name,
			ID:      id,
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		ParentId:    obj.ParentId,
		TTL:         *obj.TTL,
		Fields:      []*runtime.Field{},
		FieldByName: map[string]*runtime.Field{},
		RequiredCnt: 0,
	}

	SetDefaults_EventType(obj, et)

	et = m.generateInheritanceChain(et)

	for _, f := range obj.Fields {
		if err := isFieldExists(et.Parents, f.Name); err != nil {
			return nil, err
		}

		if rf, err := parseFieldValues(f); err != nil {
			return nil, response.ErrFieldInvalid(f.Name, f.Values)
		} else {
			et.Fields = append(et.Fields, rf)
			et.FieldByName[rf.Name] = rf
			if rf.Required {
				et.RequiredCnt++
			}
		}
	}

	if len(et.Parents) > 0 {
		for i := 0; i < len(et.Parents); i++ {
			parent := et.Parents[i]
			for _, f := range parent.Fields {
				et.FieldByName[f.Name] = f
				if f.Required {
					et.RequiredCnt++
				}
			}
		}
		et.RequiredCnt -= 2 // because BaseEvent has 2 required fields
	}

	var saved *runtime.EventType
	var err error
	if saved, err = m.saveEventType(et); err != nil {
		return nil, err
	}

	if err = m.store.Create(saved); err != nil {
		return nil, err
	}

	m.eventTypes.Store(saved.ID, saved)

	klog.V(2).InfoS("Created EventType", "id", saved.ID)
	return m.foldEventType(saved), nil
}

func (m *Manager) GetEventTypeById(id string, exploded bool) (*runtime.EventType, error) {
	if et, err := m.getEventTypeById(id, exploded); err == nil && exploded {
		return m.foldEventTypeParents([]*runtime.EventType{et})[0], nil
	} else {
		return et, err
	}
}

func (m *Manager) ListEventTypes(filter *TypeFilter, exploded bool) ([]*runtime.EventType, error) {
	if ets, err := m.listEventTypes(filter, exploded); err == nil && exploded {
		return m.foldEventTypeParents(ets), nil
	} else {
		return ets, err
	}
}

func (m *Manager) UpdateEventType(id, version string, obj *v1.EventType, old *runtime.EventType) (*runtime.EventType, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}
	deleteFields, updateFields, insertFields := generic.DifferenceAndIntersectionObjects(old.Fields, obj.Fields,
		func(value interface{}) string { return value.(*runtime.Field).Name },
		func(value interface{}) string { return value.(*v1.Field).Name })
	old.ModTime = time.Now()
	if obj.TTL != nil {
		old.TTL = *obj.TTL
	}

	cets := m.getChildEventTypes(old)

	// delete
	deleteEventTypeFields(old, deleteFields)

	for _, field := range obj.Fields {
		// whether parent or child has the field
		if err := isFieldExists(old.Parents, field.Name); err != nil {
			return nil, err
		}
		if err := isFieldExists(cets, field.Name); err != nil {
			return nil, err
		}

		newField, err := parseFieldValues(field)
		if err != nil {
			return nil, response.ErrFieldInvalid(field.Name, field.Values)
		}

		// update or insert
		if of, ok := old.FieldByName[newField.Name]; ok {
			of.Required = newField.Required
			of.Updatable = newField.Updatable
			of.Values = newField.Values
		} else {
			old.Fields = append(old.Fields, newField)
			old.FieldByName[newField.Name] = newField
		}
	}

	var saved *runtime.EventType
	var err error
	if saved, err = m.saveEventType(old); err != nil {
		return nil, err
	}

	des, _ := m.eventTypes.Load(saved.ID)
	et := des.(*runtime.EventType)
	et.TTL = saved.TTL
	et.Version = saved.Version
	et.ModTime = saved.ModTime

	deleteEventTypeFields(et, deleteFields)

	for _, name := range updateFields {
		field := et.FieldByName[name]
		newField := saved.FieldByName[field.Name]
		if field.Required && !newField.Required {
			et.RequiredCnt--
		} else if !field.Required && newField.Required {
			et.RequiredCnt++
		}
		field.Required = newField.Required
		field.Updatable = newField.Updatable
		field.Values = newField.Values
	}

	for _, name := range insertFields {
		field := saved.FieldByName[name]
		et.FieldByName[field.Name] = field
		et.Fields = append(et.Fields, field)
		if field.Required {
			et.RequiredCnt++
		}
	}

	_ = m.store.Set(et)

	m.reindexChildEventType(et, cets, insertFields, deleteFields)

	klog.V(2).InfoS("Updated EventType", "id", et.ID)
	return m.foldEventType(et), nil
}

func (m *Manager) DeleteEventType(id, ETag string) (*runtime.EventType, error) {
	if id == runtime.StandardEventTypeId || id == runtime.BaseEventTypeId {
		return nil, response.ErrDeleteFundamentalResourceFailed(id)
	}
	et, err := m.getEventTypeById(id, true)
	if err != nil {
		return nil, err
	}
	if et.Version != ETag {
		return nil, apis.ErrMismatch
	}
	if m.hasChildEventType(et.ID) {
		return nil, response.ErrChildTypeExist(et.ID)
	}

	m.etCh <- &runtime.EventType{
		ObjectMeta: meta.ObjectMeta{
			ID:      et.ID,
			Version: et.Version,
		},
	}
	res := <-m.etResCh
	if res.Err != nil {
		return nil, res.Err
	}

	m.eventTypes.Delete(et.ID)

	m.store.Delete(et)

	bucketTTL := fmt.Sprintf("%d", et.TTL)
	_ = m.jm.DeleteEventType(path.Join(bucketTTL, et.ID), time.Now().UTC())

	klog.V(2).InfoS("Deleted eventType", "id", et.ID)
	return m.foldEventType(et), nil
}

func (m *Manager) getEventTypeById(id string, exploded bool) (*runtime.EventType, error) {
	et, isExist := m.eventTypes.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	ret, _ := et.(*runtime.EventType)
	if !exploded {
		ret = m.foldEventType(ret)
	}
	return ret, nil
}

func (m *Manager) listEventTypes(filter *TypeFilter, exploded bool) ([]*runtime.EventType, error) {
	ets := make([]*runtime.EventType, 0)
	predicates := parseTypeFilter(filter)

	// descend
	byModTime := func(et1, et2 *runtime.EventType) bool { return et1.ModTime.Before(et2.ModTime) }
	sorter := ByType(byModTime)

	m.eventTypes.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.EventType)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			ets = sorter.Insert(ets, v)
		}
		return true
	})

	if !exploded {
		for i := range ets {
			ets[i] = m.foldEventType(ets[i])
		}
	}
	return ets, nil
}

func (m *Manager) generateInheritanceChain(et *runtime.EventType) *runtime.EventType {
	for t := et; t != nil && t.ParentId != nil; {
		t, _ = m.getEventTypeById(*t.ParentId, true)
		et.Parents = append(et.Parents, t)
		if strings.EqualFold(t.ID, runtime.BaseEventTypeId) || strings.EqualFold(t.ID, runtime.StandardEventTypeId) {
			break
		}
	}

	// reverse
	for i, j := 0, len(et.Parents)-1; i < j; i, j = i+1, j-1 {
		et.Parents[i], et.Parents[j] = et.Parents[j], et.Parents[i]
	}

	return et
}

func (m *Manager) foldEventTypeParents(ets []*runtime.EventType) []*runtime.EventType {
	ret := make([]*runtime.EventType, 0, len(ets))
	etById := make(map[string]*runtime.EventType, len(ets))

	for _, et := range ets {
		fold := m.foldEventType(et)
		for _, parent := range et.Parents {
			fp, ok := etById[parent.ID]
			if !ok {
				fp = m.foldEventType(parent)
				etById[fp.ID] = fp
			}
			fold.Parents = append(fold.Parents, fp)
		}
		ret = append(ret, fold)
	}

	return ret
}

func (m *Manager) foldEventType(et *runtime.EventType) *runtime.EventType {
	ret := &runtime.EventType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  et.Tenant,
			ID:      et.ID,
			Name:    et.Name,
			Version: et.Version,
			ModTime: et.ModTime,
		},
		ParentId:    et.ParentId,
		TTL:         et.TTL,
		Fields:      et.Fields,
		FieldByName: et.FieldByName,
		Parents:     nil,
	}
	return ret
}

func (m *Manager) reindexEventType() {
	m.eventTypes.Range(func(key, value interface{}) bool {
		et, _ := value.(*runtime.EventType)
		et.RequiredCnt = 0
		et.FieldByName = make(map[string]*runtime.Field, len(et.Fields))
		m.generateInheritanceChain(et)
		for _, f := range et.Fields {
			et.FieldByName[f.Name] = f
			if f.Required {
				et.RequiredCnt++
			}
		}

		if len(et.Parents) > 0 {
			for i := 0; i < len(et.Parents); i++ {
				parent := et.Parents[i]
				for _, f := range parent.Fields {
					et.FieldByName[f.Name] = f
					if f.Required {
						et.RequiredCnt++
					}
				}
			}
			et.RequiredCnt -= 2 // because BaseEvent has 2 required fields
		}

		return true
	})
}

func (m *Manager) hasChildEventType(parentId string) bool {
	ret := false
	m.eventTypes.Range(func(key, value interface{}) bool {
		et := value.(*runtime.EventType)
		if et.ParentId != nil && *et.ParentId == parentId {
			ret = true
		}
		return !ret
	})

	return ret
}

func (m *Manager) getChildEventTypes(pet *runtime.EventType) []*runtime.EventType {
	ets, _ := m.listEventTypes(&TypeFilter{parentId: pet.ID}, true)
	for i := 0; i < len(ets); i++ {
		child, _ := m.listEventTypes(&TypeFilter{parentId: ets[i].ID}, true)
		ets = append(ets, child...)
	}

	return ets
}

func (m *Manager) reindexChildEventType(parent *runtime.EventType, ets []*runtime.EventType, insertFields, deleteFields []string) {
	for _, et := range ets {
		for _, name := range insertFields {
			field := parent.FieldByName[name]
			et.FieldByName[field.Name] = field
		}

		for _, name := range deleteFields {
			delete(et.FieldByName, name)
		}

		if len(insertFields) != 0 || len(deleteFields) != 0 {
			_ = m.store.Set(et)
		}

		et.RequiredCnt = 0
		for _, field := range et.FieldByName {
			if field.Required {
				et.RequiredCnt++
			}
		}
		et.RequiredCnt -= 2 // because BaseEvent has 2 required fields
	}
}

func (m *Manager) saveEventType(et *runtime.EventType) (*runtime.EventType, error) {
	fieldByName := et.FieldByName
	et.FieldByName = nil
	parents := et.Parents
	et.Parents = nil

	m.etCh <- et
	res := <-m.etResCh
	if res.Err != nil {
		return nil, res.Err
	}

	saved := res.Saved.(*runtime.EventType)
	saved.FieldByName = fieldByName
	saved.Parents = parents

	return saved, nil
}

func (m *Manager) CreateEvent(data []byte) (string, *string, error) {
	eTag := strconv.FormatUint(randutil.Uint64n(), 10)
	var id, corId string
	var ce v1.BaseEvent

	if err := json.Unmarshal(data, &ce); err != nil {
		klog.V(3).InfoS("Failed to parse event data", "err", err)
		return "", nil, response.NewMultiError(response.ErrMalformedJSON)
	}

	if _, ok := m.things.Load(ce.ThingId); !ok {
		klog.V(3).InfoS("Thing not found", "id", ce.ThingId)
		return "", nil, response.NewMultiError(response.ErrThingNotFound(ce.ThingId))
	}

	if len(ce.Id) == 0 {
		id = uuidutil.UUID()
	} else {
		id = ce.Id
	}
	if len(ce.CorrelationId) == 0 {
		corId = uuidutil.UUID()
	} else {
		corId = ce.CorrelationId
	}

	if len(ce.TypeId) == 0 || strings.EqualFold(ce.TypeId, runtime.StandardEventTypeId) {
		var se v1.StandardEvent
		if err := json.Unmarshal(data, &se); err != nil {
			klog.V(3).InfoS("Failed to parse StandardEvent data", "err", err)
			return "", nil, response.NewMultiError(response.ErrMalformedJSON)
		}
		if err := m.validate.Struct(se); err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				klog.V(3).InfoS("Failed to validate StandardEvent data", "err", err)
				return "", nil, err
			}
			klog.V(3).InfoS("Failed to validate StandardEvent data", "err", err.(validator.ValidationErrors))
			return "", nil, err
		}
		rse := &runtime.StandardEvent{
			BaseEvent: runtime.BaseEvent{
				Id:            id,
				TypeId:        runtime.StandardEventTypeId,
				CorrelationId: corId,
				Time:          se.Time,
				ThingId:       se.ThingId,
				ETag:          eTag,
			},
			Severity:     se.Severity,
			Description:  se.Description,
			Code:         se.Code,
			Source:       se.Source,
			Acknowledged: se.Acknowledged,
		}
		et, _ := m.getEventTypeById(runtime.StandardEventTypeId, true)
		m.cacheActiveThing(rse.ThingId, runtime.StandardEventTypeId)
		if err := m.store.CreateStandardEvent(rse, et, false); err != nil {
			return "", nil, err
		}
	} else {
		et, err := m.getEventTypeById(ce.TypeId, true)
		if err != nil {
			klog.V(3).InfoS("EventType not found", "id", ce.TypeId)
			return "", nil, response.NewMultiError(response.ErrResourceNotFound(ce.TypeId))
		}

		if err = json.Unmarshal(data, &ce.Fields); err != nil {
			klog.V(3).InfoS("Failed to parse fields", "event", ce.TypeId, "err", err)
			return "", nil, response.NewMultiError(response.ErrMalformedJSON)
		}

		for _, f := range runtime.BaseEventFields.UnsortedList() {
			delete(ce.Fields, f)
		}

		requiredCnt := 0
		legalFields := map[string]interface{}{}
		for k, v := range ce.Fields {
			if f, ok := et.FieldByName[k]; ok {
				legalFields[k] = v
				if f.Required == true {
					requiredCnt++
				}
			} else {
				klog.V(4).InfoS("Ignored invalid field", "field", k)
			}
		}

		if requiredCnt != et.RequiredCnt {
			var missedRequiredFields []string
			for k, v := range et.FieldByName {
				// BaseEvent has 2 required fields
				if k == runtime.BaseEventFieldThingId || k == runtime.BaseEventFieldTime {
					continue
				}
				if _, ok := legalFields[k]; !ok && v.Required {
					missedRequiredFields = append(missedRequiredFields, k)
				}
			}
			return "", nil, response.NewMultiError(response.ErrRequiredFieldMissed(missedRequiredFields))
		}

		errs := &response.MultiError{}
		for k, v := range legalFields {
			f, _ := et.FieldByName[k]
			// https://golang.org/pkg/encoding/json/#Unmarshal
			switch f.DataType {
			case v1.DatatypeInt:
				switch v.(type) {
				case float64:
					legalFields[k] = int64(v.(float64))
				case string:
					if value, err := strconv.ParseInt(v.(string), 10, 64); err != nil {
						errs.Add(response.ErrFieldInvalid(k, v))
					} else {
						legalFields[k] = value
					}
				default:
					errs.Add(response.ErrFieldInvalid(k, v))
				}
			case v1.DatatypeDouble:
				switch v.(type) {
				case float64:
					legalFields[k] = v
				case string:
					if value, err := strconv.ParseFloat(v.(string), 64); err != nil {
						errs.Add(response.ErrFieldInvalid(k, v))
					} else {
						legalFields[k] = value
					}
				default:
					errs.Add(response.ErrFieldInvalid(k, v))
				}
			case v1.DatatypeBool:
				switch v.(type) {
				case bool:
					legalFields[k] = v
				case string:
					if value, err := strconv.ParseBool(v.(string)); err != nil {
						errs.Add(response.ErrFieldInvalid(k, v))
					} else {
						legalFields[k] = value
					}
				default:
					errs.Add(response.ErrFieldInvalid(k, v))
				}
			case v1.DatatypeString:
				fallthrough
			case v1.DatatypeUuid:
				legalFields[k] = fmt.Sprintf("%v", v)
			case v1.DatatypeEnum:
				// json unmarshal convert integer to float64
				index := fmt.Sprintf("%v", v)
				if i, err := strconv.ParseInt(index, 10, 64); err == nil {
					values := f.Values.([]interface{})
					if i < 0 || int(i) >= len(values) {
						errs.Add(response.ErrFieldInvalid(k, v))
					} else {
						legalFields[k] = values[i]
					}
				} else {
					errs.Add(response.ErrFieldInvalid(k, v))
				}
			case v1.DatatypeMap:
				key := fmt.Sprintf("%v", v)
				values := f.Values.(map[string]interface{})
				if value, ok := values[key]; ok {
					legalFields[k] = value
				} else {
					errs.Add(response.ErrFieldInvalid(k, v))
				}
			case v1.DatatypeLink:
				if l, ok := v.(string); ok {
					if _, err := url.Parse(l); err != nil {
						errs.Add(response.ErrFieldInvalid(k, v))
					} else {
						legalFields[k] = l
					}
				} else {
					errs.Add(response.ErrFieldInvalid(k, v))
				}
			case v1.DatatypeTimestamp:
				var t int64
				if f64, ok := v.(float64); ok {
					t = int64(f64)
				} else if str, ok := v.(string); ok {
					t, err = strconv.ParseInt(str, 10, 64)
					if err != nil {
						errs.Add(response.ErrFieldInvalid(k, v))
						continue
					}
				} else {
					errs.Add(response.ErrFieldInvalid(k, v))
					continue
				}
				legalFields[k] = t
			default:
				errs.Add(response.ErrFieldUnSupported(f.DataType.String()))
			}
		}

		if errs.Len() != 0 {
			return "", nil, errs
		}

		rce := &runtime.BaseEvent{
			Id:            id,
			TypeId:        ce.TypeId,
			CorrelationId: corId,
			Time:          ce.Time,
			ThingId:       ce.ThingId,
			ETag:          eTag,
		}
		m.cacheActiveThing(rce.ThingId, rce.TypeId)
		if err := m.store.CreateCustomEvent(rce, legalFields, et, false); err != nil {
			return "", nil, err
		}
	}
	return eTag, &id, nil
}

func (m *Manager) ListEvents(start, end time.Time, filter map[string]interface{}, selects []string, desc, latest bool, limit int) (interface{}, error) {
	// correct range
	now := time.Now().UTC()
	if end.IsZero() || end.After(now) {
		end = now
	}
	if start.IsZero() || start.After(now) {
		start = now.Add(-7 * 24 * time.Hour)
	}

	if start.After(end) || start.Equal(end) {
		return []map[string]interface{}{}, nil
	}

	// validate thingId
	var thing string
	activeEventTypes := sets.String{}
	if tId, ok := filter[runtime.BaseEventFieldThingId]; ok {
		thingId := tId.(string)
		if v, ok := m.things.Load(thingId); ok && v != (*sync.Map)(nil) {
			thing = thingId
			active := v.(*sync.Map)
			active.Range(func(key, value interface{}) bool {
				activeEventTypes.Insert(key.(string))
				return true
			})
		} else {
			return []map[string]interface{}{}, nil
		}
	}
	delete(filter, runtime.BaseEventFieldThingId)

	// validate typeId
	var et *runtime.EventType
	if etId, ok := filter[runtime.BaseEventFieldTypeId]; ok {
		typeId := etId.(string)
		if et, _ = m.getEventTypeById(typeId, true); et == nil {
			klog.V(4).InfoS("EventType not found, all eventTypes are considered", "id", typeId)
		}
	}
	delete(filter, runtime.BaseEventFieldTypeId)

	if len(thing) == 0 && et == nil {
		et, _ = m.getEventTypeById(runtime.StandardEventTypeId, true)
	}

	ets := make([]*runtime.EventType, 0, activeEventTypes.Len())
	if et == nil {
		for _, v := range activeEventTypes.UnsortedList() {
			if et, _ = m.getEventTypeById(v, true); et != nil {
				ets = append(ets, et)
			}
		}
	} else if activeEventTypes.Len() == 0 || activeEventTypes.Has(et.ID) {
		ets = append(ets, et)
	}

	if len(ets) == 0 {
		return []map[string]interface{}{}, nil
	}

	// validate other fields
	var legalEventTypes []*runtime.EventType
	legalFilter := make(map[string]interface{})
	if len(filter) == 0 {
		legalEventTypes = ets
	} else {
		legalEventTypesMap := make(map[*runtime.EventType]struct{}, len(ets))
		for k, v := range filter {
			for _, et := range ets {
				if f, ok := et.FieldByName[k]; ok {
					if f.Filterable {
						legalFilter[k] = v
						legalEventTypesMap[et] = struct{}{}
					} else {
						klog.V(4).InfoS("Ignored not filterable field in filter", "field", k, "typeId", et.ID)
					}
				} else {
					klog.V(4).InfoS("Ignored undefined field in filter", "filed", k, "typeId", et.ID)
				}
			}
		}

		if len(legalEventTypesMap) == 0 {
			return []map[string]interface{}{}, nil
		} else {
			legalEventTypes = make([]*runtime.EventType, 0, len(legalEventTypesMap))
			for k := range legalEventTypesMap {
				legalEventTypes = append(legalEventTypes, k)
			}
		}
	}

	// validate select fields
	legalFields := sets.String{}
	if len(selects) > 0 {
		for _, f := range selects {
			for _, et := range legalEventTypes {
				if _, ok := et.FieldByName[f]; ok {
					legalFields.Insert(f)
				} else {
					klog.V(4).InfoS("Ignored undefined filed in select", "filed", f, "typeId", et.ID)
				}
			}
		}
		legalFields.Insert(runtime.BaseEventFieldId,
			runtime.BaseEventFieldTypeId,
			runtime.BaseEventFieldCorrelationId,
			runtime.BaseEventFieldETag,
		)
	} else {
		for _, et := range legalEventTypes {
			for k := range et.FieldByName {
				legalFields.Insert(k)
			}
		}
	}

	return m.store.ListEvents(legalEventTypes, start, end, thing, legalFilter, legalFields, desc, latest, limit)
}

func (m *Manager) GetEventById(id, thingId, typeId string) (interface{}, string, error) {
	if len(thingId) > 0 {
		if active, ok := m.things.Load(thingId); !ok || active == (*sync.Map)(nil) {
			return nil, "", response.ErrThingNotFound(thingId)
		}
		// TODO check cache typeId
	}

	var et *runtime.EventType
	if et, _ = m.getEventTypeById(typeId, true); et == nil {
		if len(typeId) == 0 {
			klog.V(3).InfoS("EventType not specified, StandardEvent is honored")
		} else {
			klog.V(3).InfoS("EventType not found, StandardEvent is honored", "typeId", typeId)
		}
		et, _ = m.getEventTypeById(runtime.StandardEventTypeId, true)
	}

	e, eTag, err := m.store.GetEventById(id, thingId, et)
	if err == nil {
		return e, eTag, nil
	}
	return nil, "", err
}

func (m *Manager) DeleteEvent(id, ETag, typeId string) error {
	event, eTag, err := m.GetEventById(id, "", typeId)
	if err != nil {
		return err
	}
	if len(eTag) == 0 {
		return os.ErrNotExist
	}
	if eTag != ETag {
		return apis.ErrMismatch
	}

	data := event.(map[string]interface{})
	et, _ := m.getEventTypeById(typeId, true)
	ts := data[runtime.BaseEventFieldTime].(string)
	t, _ := time.Parse(time.RFC3339Nano, ts)

	fieldId := data[runtime.BaseEventFieldId].(string)
	thingId := data[runtime.BaseEventFieldThingId].(string)
	keys := map[string]interface{}{
		runtime.BaseEventFieldId:      fieldId,
		runtime.BaseEventFieldThingId: thingId,
	}
	if err := m.store.DeleteEvent(keys, t, et); err != nil {
		klog.V(2).InfoS("Failed to delete event", "eventId", id, "err", err)
		return nil
	}
	return nil
}

func (m *Manager) cacheActiveThing(thingId, typeId string) {
	if v, ok := m.things.Load(thingId); ok {
		active := v.(*sync.Map)
		if active == nil {
			active = &sync.Map{}
		}
		if _, ok := active.Load(typeId); !ok {
			active.Store(typeId, struct{}{})
			aet := &runtime.ActiveEventType{
				ObjectMeta: meta.ObjectMeta{
					Tenant:  security.GetTenant(),
					ID:      thingId,
					ModTime: time.Now(),
				},
				ThingId:    thingId,
				EventTypes: sets.String{},
			}
			active.Range(func(key, value interface{}) bool {
				aet.EventTypes.Insert(key.(string))
				return true
			})
			if err := m.typeStore.SaveActivateEventType(aet); err != nil {
				klog.V(3).InfoS("failed to update activate eventtype", "thingId", aet.ThingId)
			}
			m.things.Store(thingId, active)
		}
	}
}

func (m *Manager) processEvent(stopCh <-chan struct{}) {
	for {
		select {
		case obj, _ := <-m.tCh:
			t := obj.(*model.Thing)
			if t.ModTime.IsZero() {
				m.things.Delete(t.ID)
			} else if _, ok := m.things.Load(t.ID); !ok {
				m.things.Store(t.ID, (*sync.Map)(nil))
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped event watch event")
			return
		}
	}
}

func parseFieldValues(f *v1.Field) (*runtime.Field, error) {
	rf := &runtime.Field{
		Name:       f.Name,
		Filterable: f.Filterable,
		Required:   f.Required,
		Updatable:  f.Updatable,
		DataType:   f.DataType,
	}
	switch f.DataType {
	case v1.DatatypeEnum:
		var values []string
		if err := mapstructure.Decode(f.Values, &values); err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, errors.New(fmt.Sprintf("field %s's value is empty for enum", f.Name))
		}
		vis := make([]interface{}, len(values))
		for i, v := range values {
			vis[i] = v
		}
		rf.Values = vis
	case v1.DatatypeMap:
		var values map[string]string
		if err := mapstructure.Decode(f.Values, &values); err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, errors.New(fmt.Sprintf("field %s's value is empty for map", f.Name))
		}
		vis := make(map[string]interface{}, len(values))
		for k, v := range values {
			vis[k] = v
		}
		rf.Values = vis
	}
	return rf, nil
}

func deleteEventTypeFields(et *runtime.EventType, deleteFields []string) {
	i := 0
	deleteFieldSet := sets.NewString(deleteFields...)
	for _, field := range et.Fields {
		if !deleteFieldSet.Has(field.Name) {
			et.Fields[i] = field
			i++
		} else {
			if field.Required {
				et.RequiredCnt--
			}
			delete(et.FieldByName, field.Name)
		}
	}
	for j := i; j < len(et.Fields); j++ {
		et.Fields[j] = nil
	}
	et.Fields = et.Fields[:i]
}

func isFieldExists(ets []*runtime.EventType, name string) error {
	for _, et := range ets {
		for _, field := range et.Fields {
			if field.Name == name {
				return response.ErrFieldExists(name, et.ID)
			}
		}
	}
	return nil
}
