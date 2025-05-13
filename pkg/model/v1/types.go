package v1

import "time"

type Datatype byte

const (
	DataTypeString Datatype = iota
	DataTypeInt
	DataTypeLong
	DataTypeDouble
	DataTypeBoolean
)

var DataTypeToString = map[Datatype]string{
	DataTypeString:  "string",
	DataTypeInt:     "int",
	DataTypeLong:    "long",
	DataTypeDouble:  "double",
	DataTypeBoolean: "bool",
}

var DataTypeFromString = map[string]Datatype{
	"string": DataTypeString,
	"int":    DataTypeInt,
	"long":   DataTypeLong,
	"double": DataTypeDouble,
	"bool":   DataTypeBoolean,
}

func (dt Datatype) String() string {
	return DataTypeToString[dt]
}

type AccessMode byte

const (
	AccessModeReadOnly  = 'r'
	AccessModeReadWrite = 'r' + 'w'
)

// var AccessModeToString = map[AccessMode]string {
//    AccessModeReadOnly:  "r",
//    AccessModeReadWrite: "rw",
// }
//
// var AccessModeFromString = map[string]AccessMode{
//    "r":  AccessModeReadOnly,
//    "rw": AccessModeReadWrite,
// }

type AgentDataType byte

const (
	AgentDataTypeTimeSeries AgentDataType = iota
	AgentDataTypeEvent
	AgentDataTypeCtrlAck
)

var AgentDataTypeToString = map[AgentDataType]string{
	AgentDataTypeTimeSeries: "timeSeries",
	AgentDataTypeEvent:      "event",
	AgentDataTypeCtrlAck:    "ack",
}

var AgentDataTypeFromString = map[string]AgentDataType{
	"timeSeries": AgentDataTypeTimeSeries,
	"event":      AgentDataTypeEvent,
	"ack":        AgentDataTypeCtrlAck,
}

type Agent struct {
	ThingId     string        `json:"thingId" binding:"required,len=32"`
	Name        string        `json:"name" binding:"required,min=1,max=64,excludesall=\u002F\u005C"`
	Description *string       `json:"description" binding:"omitempty,max=255"`
	TypeId      *string       `json:"typeId" binding:"omitempty,max=80"`
	DataSources []*DataSource `json:"dataSources" binding:"dive"`
}

type DataPoint struct {
	Id          string       `json:"id" binding:"required,min=1,max=64,excludesall=\u002F\u005C"`
	Name        string       `json:"name" binding:"min=0,max=64,excludesall=\u002F\u005C"`
	Description *string      `json:"description" binding:"omitempty,max=255"`
	Datatype    Datatype     `json:"datatype"`
	Unit        string       `json:"unit"`
	AccessMode  AccessMode   `json:"accessMode"`
	CustomData  *interface{} `json:"customData,omitempty"`
}

type DataSource struct {
	Name        string       `json:"name" binding:"required,min=1,max=64,excludesall=\u002F\u005C"`
	Description *string      `json:"description" binding:"omitempty,max=255"`
	DataPoints  []*DataPoint `json:"dataPoints" binding:"dive"`
	CustomData  *interface{} `json:"customData,omitempty"`
}

type AgentType struct {
	Name        string        `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u003F\u005C"`
	Description *string       `json:"description" binding:"omitempty,max=255"`
	DataSources []*DataSource `json:"dataSources" binding:"dive"`
}

type Characteristic struct {
	Name         string   `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u005C"`
	Unit         string   `json:"unit,omitempty"`
	Length       int      `json:"length,omitempty"`
	DataType     Datatype `json:"datatype"`
	DefaultValue string   `json:"defaultValue,omitempty"`
	Searchable   bool     `json:"searchable"`
}

type Property struct {
	Name       string     `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u005C"`
	Unit       string     `json:"unit,omitempty"`
	Length     int        `json:"length,omitempty"`
	Datatype   Datatype   `json:"datatype"`
	AccessMode AccessMode `json:"accessMode"`
	Min        *float64   `json:"min,omitempty"`
	Max        *float64   `json:"max,omitempty"`
}

type PropertySetType struct {
	Name        string      `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u003F\u005C"`
	Description *string     `json:"description" binding:"omitempty,max=255"`
	Properties  []*Property `json:"properties" binding:"required,dive"`
}

type PropertySet struct {
	Name              string  `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u003F\u005C"`
	PropertySetTypeId *string `json:"propertySetTypeId,omitempty"`
	// it is same with PropertySetType, but they have different validator
	PropertySetType *PropertySetType `json:"propertySetType,omitempty"`
	// Description *string     `json:"description" binding:"omitempty,max=255"`
	// Properties  []*Property `json:"properties" binding:"dive,required_without=PropertySetTypeId"`
}

type ThingType struct {
	Name            string            `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u003F\u005C"`
	Description     *string           `json:"description" binding:"omitempty,max=255"`
	ParentTypeId    string            `json:"parentTypeId,omitempty"`
	Instantiable    *bool             `json:"instantiable,omitempty"`
	Characteristics []*Characteristic `json:"characteristics,omitempty" binding:"dive"`
	PropertySets    []*PropertySet    `json:"propertySets,omitempty" binding:"dive"`
}

type Value struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

type Thing struct {
	Name        string  `json:"name" binding:"required,min=1,max=64,excludesall=\u002F\u005C"`
	Description *string `json:"description" binding:"omitempty,max=255"`
	// TODO adapt with front end
	// geographic address for the thing
	// Location    GeoAddress `json:location`
	TypeId          string  `json:"typeId" binding:"required"`
	ParentId        string  `json:"parentId,omitempty"`
	TimeZone        *string `json:"timeZone,omitempty"`
	Characteristics []Value `json:"characteristics,omitempty" binding:"dive"`
}

type DataPointMapping struct {
	AgentId         string `json:"agentId" binding:"required"`
	DataSourceId    string `json:"dataSourceId" binding:"required"`
	DataPointId     string `json:"dataPointId" binding:"required"`
	ThingId         string `json:"thingId" binding:"required"`
	PropertySetName string `json:"propertySetName" binding:"required"`
	PropertyName    string `json:"propertyName" binding:"required"`
}

// up link

type DataPointValue struct {
	DataPointId string `json:"dataPointId" binding:"required"`
	// Value       string `json:"value" binding:"required"`
	Value interface{} `json:"value" binding:"required"`
}

type EventData struct {
	Id            string                 `json:"id,omitempty"`
	CorrelationId string                 `json:"correlationId,omitempty"`
	Timestamp     time.Time              `json:"timestamp" binding:"required"`
	Type          string                 `json:"type,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

type TimeSeriesData struct {
	Timestamp time.Time        `json:"timestamp" binding:"required"`
	Values    []DataPointValue `json:"values" binding:"required,dive"`
}

type AgentPayload struct {
	Type            AgentDataType `json:"type,omitempty"`
	Version         *string       `json:"version,omitempty"`
	DataSourceId    *string       `json:"dataSourceId,omitempty"`
	PropertySetName *string       `json:"propertySetName,omitempty"`
	// array of concreted data
	Data interface{} `json:"data" binding:"required"`
}

type AgentData struct {
	// Type    string       `json:"type,omitempty"`
	// Version string       `json:"version,omitempty"`
	Payload AgentPayload `json:"payload" binding:"required,dive"`
}

// down link

type ControlData struct {
	Seq     int64                  `json:"seq" binding:"required"`
	ThingId string                 `json:"thingId" binding:"required"`
	PsName  string                 `json:"PropertySetName,omitempty"`
	TypeId  string                 `json:"typeId,omitempty"`
	Ack     bool                   `json:"ack,omitempty"`
	Timeout time.Duration          `json:"timeout,omitempty"`
	Time    time.Time              `json:"_time" binding:"required"`
	Options map[string]interface{} `json:"options" binding:"required_without=TypeId"`
}

type AckData struct {
	Seq       string    `json:"seq" binding:"required"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Code      int       `json:"code,required"`
	Message   *string   `json:"message,omitempty"`
}
