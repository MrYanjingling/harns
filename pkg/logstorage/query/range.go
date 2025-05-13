package query

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/logstorage/data"
	"math"
	"time"
)

type TableDefinitions []*data.TableDefinition

type ReplaceTableNameFun func() string

type Range struct {
	TableDefinitions
	*data.Range
	selects sets.String
	keys    map[string]interface{}
	limit   int
	desc    bool
	byTime  bool // if true, the result is grouped by time. If false, the result is grouped by column name

	replaceTableNameFun ReplaceTableNameFun
}

func NewRange(tds []*data.TableDefinition, start, end time.Time, selects sets.String, keys map[string]interface{}, limit int, desc, byTime bool, f ReplaceTableNameFun) *Range {
	return &Range{
		TableDefinitions: tds,
		Range:            data.NewRange(start, end),
		selects:          selects,
		keys:             keys,
		limit:            limit,
		desc:             desc,
		byTime:           byTime,

		replaceTableNameFun: f,
	}
}

func (r *Range) GetKeys() map[string]interface{} {
	return r.keys
}

func (r *Range) GetSelects() sets.String {
	return r.selects
}

func (r *Range) GetLimit() int {
	return r.limit
}

func (r *Range) IsDesc() bool {
	return r.desc
}

func (r *Range) ByTime() bool {
	return r.byTime
}

func (r *Range) ReplaceTableName() ReplaceTableNameFun {
	return r.replaceTableNameFun
}

func (tds TableDefinitions) GetTableNames() []string {
	var names []string
	for _, td := range tds {
		names = append(names, td.GetTableName())
	}
	return names
}

func (tds TableDefinitions) GetTableDefinition(table string) *data.TableDefinition {
	for i := 0; i < len(tds); i++ {
		if tds[i].GetTableName() == table {
			return tds[i]
		}
	}
	return nil
}

// GetTTL
// the caller should make sure all tables have the same TTL for one operation
func (tds TableDefinitions) GetTTL() int {
	max := math.MaxInt
	for _, td := range tds {
		if ttl, ok := td.GetTableProperty(data.TTL).(int); ok {
			if max == math.MaxInt {
				max = ttl
			} else if ttl != max {
				// Do NOT support query cross ttl
				klog.InfoS("Invalid query", "ttl", ttl)
			}
		}
	}
	return max
}

func (tds TableDefinitions) IsPrimaryKey(columnName string) bool {
	for _, td := range tds {
		if td.IsPrimaryKey(columnName) {
			return true
		}
	}
	return false
}

func (tds TableDefinitions) IsSecondaryKey(columnName string) bool {
	for _, td := range tds {
		if td.IsSecondaryKey(columnName) {
			return true
		}
	}
	return false
}

func (tds TableDefinitions) IsAttribute(columnName string) bool {
	for _, td := range tds {
		if td.IsAttribute(columnName) {
			return true
		}
	}
	return false
}

func (tds TableDefinitions) GetColumnDatatype(columnName string) data.ColumnDatatype {
	for _, td := range tds {
		if datatype := td.GetColumnDatatype(columnName); datatype != nil {
			return *datatype
		}
	}
	return data.ColumnDatatypeString
}

// TODO
func (tds TableDefinitions) GetTableProperty(key string) interface{} {
	return tds[0].GetTableProperty(key)
}
