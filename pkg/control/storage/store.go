package storage

import (
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/control/v1"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	model "lightiot/pkg/model/runtime"
	mv1 "lightiot/pkg/model/v1"
	"time"
)

type Store struct {
	logStore logstorage.Interface
	ttl      int
	cache    *lru.Cache
	stopCh   <-chan struct{}
}

func NewStore(stopCh <-chan struct{}, logStore logstorage.Interface, ttl int) *Store {
	cache, err := lru.New(cacheSize)
	if err != nil {
		klog.InfoS("Failed to create cache", "err", err)
	}
	return &Store{
		logStore: logStore,
		ttl:      ttl,
		cache:    cache,
		stopCh:   stopCh,
	}
}

func (s *Store) Close() {
	// it shall flush async point
	s.logStore.Close()
}

func (s *Store) CreateCommand(cmd *runtime.Command, ct *runtime.CommandType) {
	td := s.getCommandTableDefinition(ct)

	keys := map[string]interface{}{
		runtime.CommandFieldThingId: cmd.ThingId,
		runtime.CommandFieldSeq:     cmd.Seq,
	}
	row := &data.Row{}
	row.SetValue(runtime.CommandFieldTime, cmd.Time)
	row.SetValue(runtime.CommandFieldState, cmd.State)
	row.SetValue(runtime.CommandFieldCode, cmd.Code)
	if cmd.Message != nil && len(*cmd.Message) != 0 {
		row.SetValue(runtime.CommandFieldMsg, *cmd.Message)
	}

	for k, v := range cmd.Options {
		if o, _ := ct.OptionByName[k]; o.Filterable {
			keys[k] = v
		} else {
			row.SetValue(k, v)
		}
	}

	statement := insert.NewUpsert(td, keys, []*data.Row{row}, false)

	_, _ = s.logStore.Insert(statement)
}

func (s *Store) ListCommands(thingId string, commandTypes []*runtime.CommandType, start, end time.Time, desc, latest bool, limit int) (interface{}, error) {
	var tds []*data.TableDefinition
	selects := sets.NewString(fixedCmdSelects...)
	for _, ct := range commandTypes {
		tds = append(tds, s.getCommandTableDefinition(ct))
		for k := range ct.OptionByName {
			selects.Insert(k)
		}
	}
	keys := map[string]interface{}{
		runtime.CommandFieldThingId: thingId,
	}
	if latest {
		q := query.NewSingle(tds, selects, keys, latest)
		return s.logStore.Get(q)
	} else {
		q := query.NewRange(tds, start, end, selects, keys, limit, desc, true, func() string { return runtime.CommandFieldTypeId })
		return s.logStore.List(q)
	}
}

func (s *Store) ListCommandsWithFilter(thingId string, commandType *runtime.CommandType, filter map[string]interface{}, start, end time.Time, desc, latest bool, limit int) (interface{}, error) {
	tds := []*data.TableDefinition{s.getCommandTableDefinition(commandType)}
	selects := sets.NewString(fixedCmdSelects...)
	for k := range commandType.OptionByName {
		selects.Insert(k)
	}
	keys := map[string]interface{}{
		runtime.CommandFieldThingId: thingId,
	}
	for k, v := range filter {
		keys[k] = v
	}

	if latest {
		q := query.NewSingle(tds, selects, keys, latest)
		return s.logStore.Get(q)
	} else {
		q := query.NewRange(tds, start, end, selects, keys, limit, desc, true, func() string { return runtime.CommandFieldTypeId })
		return s.logStore.List(q)
	}
}

func (s *Store) getCommandTableDefinition(ct *runtime.CommandType) *data.TableDefinition {
	v, ok := s.cache.Get(fmt.Sprintf("%s.%s", ct.ThingTypeId, ct.ID))
	if ok {
		return v.(*data.TableDefinition)
	}
	td := data.NewTableDefinition(ct.ID, make(map[string]*data.Column, 6+len(ct.Options)), map[string]interface{}{data.Database: cmdRaw})

	td.AddColumn(runtime.CommandFieldThingId, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
		AddColumn(runtime.CommandFieldSeq, data.ColumnTypeSecondaryKey, data.ColumnDatatypeInt).
		AddColumn(runtime.CommandFieldTime, data.ColumnTypeAttribute, data.ColumnDatatypeTimestamp).
		AddColumn(runtime.CommandFieldState, data.ColumnTypeAttribute, data.ColumnDatatypeString).
		AddColumn(runtime.CommandFieldCode, data.ColumnTypeAttribute, data.ColumnDatatypeInt).
		AddColumn(runtime.CommandFieldMsg, data.ColumnTypeAttribute, data.ColumnDatatypeString)

	for _, o := range ct.Options {
		if o.Filterable {
			td.AddColumn(o.Name, data.ColumnTypeSecondaryKey, convertOptionDatatype(o.Datatype))
		} else {
			td.AddColumn(o.Name, data.ColumnTypeAttribute, convertOptionDatatype(o.Datatype))
		}
	}
	s.cache.Add(fmt.Sprintf("%s.%s", ct.ThingTypeId, ct.ID), td)
	return td
}

func (s *Store) CreateAction(act *runtime.Action, ps *model.PropertySet) {
	td := s.getActionTableDefinition(ps)

	keys := map[string]interface{}{
		runtime.ActionFieldThingId: act.ThingId,
		runtime.ActionFieldPsName:  act.PsName,
		runtime.ActionFieldSeq:     act.Seq,
	}
	row := &data.Row{}
	row.SetValue(runtime.ActionFieldTime, act.Time)
	row.SetValue(runtime.ActionFieldState, act.State)
	row.SetValue(runtime.ActionFieldCode, act.Code)
	if act.Message != nil && len(*act.Message) != 0 {
		row.SetValue(runtime.ActionFieldMsg, act.Message)
	}

	for k, v := range act.Actions {
		row.SetValue(k, v)
	}

	statement := insert.NewUpsert(td, keys, []*data.Row{row}, false)

	_, _ = s.logStore.Insert(statement)
}

func (s *Store) ListActions(thingId string, ps *model.PropertySet, start, end time.Time, desc, latest bool, limit int) (interface{}, error) {
	td := s.getActionTableDefinition(ps)
	selects := sets.NewString(fixedActionSelects...)
	for _, p := range ps.PropertySetType.Properties {
		selects.Insert(p.Name)
	}
	keys := map[string]interface{}{
		runtime.ActionFieldThingId: thingId,
		runtime.ActionFieldPsName:  ps.Name,
	}
	if latest {
		q := query.NewSingle([]*data.TableDefinition{td}, selects, keys, latest)
		return s.logStore.Get(q)
	} else {
		q := query.NewRange([]*data.TableDefinition{td}, start, end, selects, keys, limit, desc, true, nil)
		return s.logStore.List(q)
	}
}

func (s *Store) getActionTableDefinition(ps *model.PropertySet) *data.TableDefinition {
	v, ok := s.cache.Get(ps.PropertySetType.ID)
	if ok {
		return v.(*data.TableDefinition)
	}

	td := data.NewTableDefinition(ps.PropertySetType.ID, make(map[string]*data.Column, 8+len(ps.PropertySetType.Properties)), map[string]interface{}{data.Database: actionRaw})
	td.AddColumn(runtime.ActionFieldThingId, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
		AddColumn(runtime.ActionFieldPsName, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
		AddColumn(runtime.ActionFieldSeq, data.ColumnTypeSecondaryKey, data.ColumnDatatypeInt).
		AddColumn(runtime.ActionFieldTime, data.ColumnTypeAttribute, data.ColumnDatatypeTimestamp).
		AddColumn(runtime.ActionFieldState, data.ColumnTypeAttribute, data.ColumnDatatypeString).
		AddColumn(runtime.ActionFieldCode, data.ColumnTypeAttribute, data.ColumnDatatypeInt).
		AddColumn(runtime.ActionFieldMsg, data.ColumnTypeAttribute, data.ColumnDatatypeString)
	for _, p := range ps.PropertySetType.Properties {
		td.AddColumn(p.Name, data.ColumnTypeAttribute, convertPropertyDatatype(p.DataType))
	}
	s.cache.Add(ps.PropertySetType.ID, td)
	return td
}

func convertOptionDatatype(t v1.Datatype) data.ColumnDatatype {
	switch t {
	case v1.DatatypeInt:
		return data.ColumnDatatypeInt
	case v1.DatatypeDouble:
		return data.ColumnDatatypeDouble
	case v1.DatatypeBool:
		return data.ColumnDatatypeBoolean
	case v1.DatatypeTimestamp:
		return data.ColumnDatatypeTimestamp
	default:
		return data.ColumnDatatypeString
	}
}

func convertPropertyDatatype(t mv1.Datatype) data.ColumnDatatype {
	switch t {
	case mv1.DataTypeInt:
		return data.ColumnDatatypeInt
	case mv1.DataTypeLong:
		return data.ColumnDatatypeLong
	case mv1.DataTypeDouble:
		return data.ColumnDatatypeDouble
	case mv1.DataTypeBoolean:
		return data.ColumnDatatypeBoolean
	default:
		return data.ColumnDatatypeString
	}
}
