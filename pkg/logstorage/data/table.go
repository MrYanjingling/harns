package data

import "math"

type StoreType byte

const (
	StoreTypeInfluxDB StoreType = iota
	StoreTypeClickHouse
)

type TableDefinition struct {
	name       string
	columns    map[string]*Column
	properties map[string]interface{}
	storeType  StoreType
}

func NewTableDefinition(name string, columns map[string]*Column, properties map[string]interface{}) *TableDefinition {
	if columns == nil {
		columns = make(map[string]*Column)
	}
	return &TableDefinition{
		name:       name,
		columns:    columns,
		properties: properties,
	}
}

func (td *TableDefinition) GetTableName() string {
	return td.name
}

func (td *TableDefinition) TableName(tableName string) {
	td.name = tableName
}

func (td *TableDefinition) GetTableProperty(key string) interface{} {
	return td.properties[key]
}

func (td *TableDefinition) AddColumn(name string, columnType ColumnType, datatype ColumnDatatype) *TableDefinition {
	td.columns[name] = NewColumn(name, columnType, datatype)
	return td
}

func (td *TableDefinition) AddTableProperty(key string, value interface{}) *TableDefinition {
	td.properties[key] = value
	return td
}

func (td *TableDefinition) IsPrimaryKey(columnName string) bool {
	if c, ok := td.columns[columnName]; ok {
		return c.columnType == ColumnTypePrimaryKey
	}
	return false
}

func (td *TableDefinition) IsSecondaryKey(columnName string) bool {
	if c, ok := td.columns[columnName]; ok {
		return c.columnType == ColumnTypeSecondaryKey
	}
	return false
}

func (td *TableDefinition) IsAttribute(columnName string) bool {
	if c, ok := td.columns[columnName]; ok {
		return c.columnType == ColumnTypeAttribute
	}
	return false
}

func (td *TableDefinition) GetColumnDatatype(columnName string) *ColumnDatatype {
	if c, ok := td.columns[columnName]; ok {
		return &c.datatype
	}
	return nil
}

func (td *TableDefinition) GetTTL() int {
	if ttl, ok := td.properties[TTL].(int); ok {
		return ttl
	}
	return math.MaxInt
}

func (td *TableDefinition) GetColumns() map[string]*Column {
	return td.columns
}
