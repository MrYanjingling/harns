package data

import "time"

var (
	DefaultColumns = []*Column{
		{
			name:       "it",
			columnType: ColumnTypeAttribute,
			datatype:   ColumnDatatypeTimestamp,
			order:      ColumnOrderDESC,
		},
	}
)

type ColumnType byte

const (
	ColumnTypePrimaryKey ColumnType = iota
	ColumnTypeSecondaryKey
	ColumnTypeUniqueKey
	ColumnTypeAttribute
)

type ColumnDatatype byte

const (
	ColumnDatatypeString ColumnDatatype = iota
	ColumnDatatypeInt
	ColumnDatatypeLong
	ColumnDatatypeDouble
	ColumnDatatypeBoolean
	ColumnDatatypeTimestamp
)

type ColumnOrder byte

const (
	ColumnOrderDESC ColumnOrder = iota
	ColumnOrderASC
	ColumnOrderNone
)

type Column struct {
	name       string
	columnType ColumnType
	datatype   ColumnDatatype
	precision  int
	scale      int
	length     int
	order      ColumnOrder
}

func NewColumn(name string, columnType ColumnType, datatype ColumnDatatype) *Column {
	return &Column{
		name:       name,
		columnType: columnType,
		datatype:   datatype,
	}
}

func (c *Column) GetName() string {
	return c.name
}

func (c *Column) GetColumnType() ColumnType {
	return c.columnType
}

func (c *Column) GetDatatype() ColumnDatatype {
	return c.datatype
}

func (c *Column) GetPrecision() int {
	return c.precision
}

func (c *Column) GetScale() int {
	return c.scale
}

func (c *Column) GetLength() int {
	return c.length
}

func (c *Column) GetOrder() ColumnOrder {
	return c.order
}

type ColumnValue struct {
	column *Column
	value  interface{}
}

func (cv *ColumnValue) getValue() interface{} {
	return cv.value
}

type ResultColumn struct {
	Value interface{}
	Time  time.Time
}
