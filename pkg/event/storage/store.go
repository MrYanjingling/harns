package storage

import (
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/event/runtime"
	"lightiot/pkg/event/v1"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"time"
)

type Store struct {
	logStorage logstorage.Interface
	cache      *lru.Cache
	stopCh     <-chan struct{}
}

func NewStore(stopCh <-chan struct{}, logStorage logstorage.Interface) *Store {
	cache, err := lru.New(cacheSize)
	if err != nil {
		klog.InfoS("Failed to create cache", "err", err)
	}

	return &Store{
		logStorage: logStorage,
		cache:      cache,
		stopCh:     stopCh,
	}
}

func convertDataType(dt v1.Datatype) data.ColumnDatatype {
	var cdt data.ColumnDatatype
	switch dt {
	case v1.DatatypeString, v1.DatatypeEnum, v1.DatatypeMap, v1.DatatypeUuid, v1.DatatypeLink:
		cdt = data.ColumnDatatypeString
	case v1.DatatypeBool:
		cdt = data.ColumnDatatypeBoolean
	case v1.DatatypeDouble:
		cdt = data.ColumnDatatypeDouble
	case v1.DatatypeInt:
		cdt = data.ColumnDatatypeInt
	case v1.DatatypeTimestamp:
		cdt = data.ColumnDatatypeTimestamp
	}

	return cdt
}

func (s *Store) getTableDefinition(et *runtime.EventType) *data.TableDefinition {
	if v, ok := s.cache.Get(et.ID); ok {
		return v.(*data.TableDefinition)
	}

	return s.newTableDefinition(et)
}

func (s *Store) newTableDefinition(et *runtime.EventType) *data.TableDefinition {
	td := data.NewTableDefinition(et.ID, map[string]*data.Column{}, map[string]interface{}{data.Database: eventRawBucket /*, data.TTL: et.TTL*/})

	for _, field := range et.FieldByName {
		ct := data.ColumnTypeAttribute
		if field.Name == runtime.BaseEventFieldThingId {
			ct = data.ColumnTypePrimaryKey
		} else if field.Filterable {
			ct = data.ColumnTypeSecondaryKey
		}

		td.AddColumn(field.Name, ct, convertDataType(field.DataType))
	}

	return td
}

func (s *Store) Create(et *runtime.EventType) error {
	td := s.getTableDefinition(et)
	if err := s.logStorage.Create(td); err != nil {
		return err
	}

	s.cache.Add(et.ID, td)
	return nil
}

func (s *Store) Set(et *runtime.EventType) error {
	// TODO update database schema and TTL
	s.cache.Add(et.ID, s.newTableDefinition(et))
	return nil
}

func (s *Store) Delete(et *runtime.EventType) {
	s.cache.Remove(et.ID)
}

func (s *Store) Close() {
	// it shall flush async point
	s.logStorage.Close()
}

func (s *Store) CreateStandardEvent(se *runtime.StandardEvent, et *runtime.EventType, sync bool) error {
	timestamp, err := time.Parse(time.RFC3339Nano, se.Time)
	if err != nil {
		klog.V(3).InfoS("Failed to parse time", "time", se.Time, "err", err)
		return response.NewMultiError(response.ErrTimestampInvalid(se.Time))
	}

	var keys map[string]interface{}
	td := s.getTableDefinition(et)

	row := data.NewRow(map[string]interface{}{
		runtime.BaseEventFieldTime:          timestamp.Add(time.Duration(et.TTL-defaultEventTTL) * 24 * time.Hour),
		runtime.BaseEventFieldFireTime:      timestamp,
		runtime.BaseEventFieldId:            se.Id,
		runtime.BaseEventFieldCorrelationId: se.CorrelationId,
		runtime.BaseEventFieldETag:          se.ETag,
		runtime.BaseEventFieldTypeId:        runtime.StandardEventTypeId,

		runtime.StandardEventFieldSeverity:     se.Severity,
		runtime.StandardEventFieldDescription:  se.Description,
		runtime.StandardEventFieldCode:         se.Code,
		runtime.StandardEventFieldSource:       se.Source,
		runtime.StandardEventFieldAcknowledged: se.Acknowledged,
	})

	// TODO clickhouse
	keys = map[string]interface{}{
		runtime.BaseEventFieldThingId: se.ThingId,
	}

	statement := insert.NewUpsert(td, keys, []*data.Row{row}, sync)
	_, _ = s.logStorage.Insert(statement)

	return nil
}

func (s *Store) CreateCustomEvent(ce *runtime.BaseEvent, fields map[string]interface{}, et *runtime.EventType, sync bool) error {
	timestamp, err := time.Parse(time.RFC3339Nano, ce.Time)
	if err != nil {
		klog.V(3).InfoS("Failed to parse time", "time", ce.Time, "err", err)
		return response.NewMultiError(response.ErrTimestampInvalid(ce.Time))
	}

	var keys map[string]interface{}
	td := s.getTableDefinition(et)

	row := data.NewRow(map[string]interface{}{
		runtime.BaseEventFieldTime:          timestamp.Add(time.Duration(et.TTL-defaultEventTTL) * 24 * time.Hour),
		runtime.BaseEventFieldId:            ce.Id,
		runtime.BaseEventFieldCorrelationId: ce.CorrelationId,
		runtime.BaseEventFieldETag:          ce.ETag,
		runtime.BaseEventFieldTypeId:        ce.TypeId,
		runtime.BaseEventFieldFireTime:      timestamp,
	})

	// TODO clickhouse
	keys = map[string]interface{}{
		runtime.BaseEventFieldThingId: ce.ThingId,
	}

	for k, v := range fields {
		row.SetValue(k, v)
	}

	statement := insert.NewUpsert(td, keys, []*data.Row{row}, sync)
	_, _ = s.logStorage.Insert(statement)

	return nil
}

func (s *Store) ListEvents(ets []*runtime.EventType, start, end time.Time, thing string, legalFilter map[string]interface{}, legalFields sets.String, desc, latest bool, limit int) (interface{}, error) {
	maxTTL := 0
	tds := make([]*data.TableDefinition, 0, len(ets))
	for _, et := range ets {
		tds = append(tds, s.getTableDefinition(et))
		if et.TTL > maxTTL {
			maxTTL = et.TTL
		}
	}

	if len(thing) > 0 {
		legalFilter[runtime.BaseEventFieldThingId] = thing
	}

	var (
		ret interface{}
		err error
	)
	legalFields.Insert(runtime.BaseEventFieldFireTime)
	if latest {
		q := query.NewSingleWithEndTime(tds, time.Now().Add(time.Duration(maxTTL-defaultEventTTL)*24*time.Hour), legalFields, legalFilter, latest)
		ret, err = s.logStorage.Get(q)
	} else {
		shift := time.Duration(maxTTL-defaultEventTTL) * 24 * time.Hour
		q := query.NewRange(tds, start.Add(shift), end.Add(shift), legalFields, legalFilter, limit, desc, true, nil)
		ret, err = s.logStorage.List(q)
	}
	es := ret.([]map[string]interface{})
	for _, e := range es {
		if t, ok := e[runtime.BaseEventFieldFireTime]; ok {
			e[runtime.BaseEventFieldTime] = t
			delete(e, runtime.BaseEventFieldFireTime)
		}
	}

	return es, err
}

func (s *Store) GetEventById(id string, thing string, et *runtime.EventType) (interface{}, string, error) {
	td := s.getTableDefinition(et)

	selects := sets.String{}
	for name, _ := range et.FieldByName {
		selects.Insert(name)
	}

	keys := map[string]interface{}{
		runtime.BaseEventFieldId: id,
	}

	if len(thing) > 0 {
		keys[runtime.BaseEventFieldThingId] = thing
	}

	selects.Insert(runtime.BaseEventFieldFireTime)
	q := query.NewSingleWithEndTime([]*data.TableDefinition{td}, time.Now().Add(time.Duration(et.TTL-defaultEventTTL)*24*time.Hour), selects, keys, false)
	ret, err := s.logStorage.Get(q)
	if err != nil {
		return nil, "", err
	}

	d := ret.([]map[string]interface{})
	if len(d) > 0 {
		if t, ok := d[0][runtime.BaseEventFieldFireTime]; ok {
			d[0][runtime.BaseEventFieldTime] = t
			delete(d[0], runtime.BaseEventFieldFireTime)
		}
		return d[0], fmt.Sprintf("%v", d[0]["eTag"]), nil
	} else {
		return map[string]interface{}{}, "", nil
	}
}

func (s *Store) DeleteEvent(keys map[string]interface{}, t time.Time, et *runtime.EventType) error {
	td := s.getTableDefinition(et)
	shift := time.Duration(et.TTL-s.logStorage.GetTTLDays(eventRawBucket)) * 24 * time.Hour

	statement := insert.NewDelete(td, t.Add(shift), t.Add(shift), keys)
	if _, err := s.logStorage.Delete(statement); err != nil {
		return err
	}
	return nil
}

func (s *Store) GetLogStorage() logstorage.Interface {
	return s.logStorage
}
