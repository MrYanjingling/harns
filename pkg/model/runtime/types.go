package runtime

import (
	"lightiot/pkg/generic/meta"
	v1 "lightiot/pkg/model/v1"
	"strings"
	"time"
)

// TimeZone TODO it is an ugly workaround, gob cannot marshal time.Location
// https://github.com/golang/go/issues/5819
// https://www.reddit.com/r/golang/comments/g0hxnw/how_to_preserve_time_localelocation_when/
type TimeZone time.Location

type ThingType struct {
	meta.ObjectMeta
	Description     *string           `json:"description,omitempty"`
	ParentTypeId    string            `json:"parentTypeId,omitempty"`
	Instantiable    *bool             `json:"instantiable"`
	Characteristics []*Characteristic `json:"characteristics,omitempty"`
	PropertySets    []*PropertySet    `json:"propertySets,omitempty"`

	CharacteristicByName map[string]*Characteristic `json:"-"`
	PropertySetByName    map[string]*PropertySet    `json:"-"`
}

type Value struct {
	Name       string          `json:"name"`
	Value      string          `json:"value"`
	Definition *Characteristic `json:"-"`
}

type Thing struct {
	meta.ObjectMeta
	Description     *string    `json:"description,omitempty"`
	Type            *ThingType `json:"-"`
	Parent          *Thing     `json:"-"`
	TimeZone        *TimeZone  `json:"timeZone"`
	Characteristics []Value    `json:"characteristics,omitempty"`

	CharacteristicByName map[string]*Characteristic `json:"-"`
	PropertySetByName    map[string]*PropertySet    `json:"-"`
}

type Characteristic struct {
	Name         string      `json:"name"`
	Unit         string      `json:"unit"`
	Length       int         `json:"length"`
	DataType     v1.Datatype `json:"datatype"`
	DefaultValue string      `json:"defaultValue"`
	Searchable   bool        `json:"searchable"`
}

type Property struct {
	Name       string        `json:"name"`
	Unit       string        `json:"unit"`
	Length     int           `json:"length"`
	DataType   v1.Datatype   `json:"datatype"`
	AccessMode v1.AccessMode `json:"accessMode"`
	Min        *float64      `json:"min,omitempty"`
	Max        *float64      `json:"max,omitempty"`
}

type PropertySet struct {
	Name            string           `json:"name"`
	PropertySetType *PropertySetType `json:"propertySetType"`
}

type PropertySetType struct {
	meta.ObjectMeta
	Description *string     `json:"description,omitempty"`
	Properties  []*Property `json:"properties"`

	PropertyByName map[string]*Property `json:"-"`
}

func (pst *PropertySetType) IsShared() bool {
	return strings.Contains(pst.ID, ".")
}

type Agent struct {
	meta.ObjectMeta
	ThingId          string                 `json:"thingId"`
	Description      *string                `json:"description,omitempty"`
	TypeId           *string                `json:"typeId,omitempty"`
	DataSources      []*DataSource          `json:"dataSources"`
	Onboard          string                 `json:"onboard"`
	Online           bool                   `json:"online"`
	DataSourceByName map[string]*DataSource `json:"-"`
}

type DataPoint struct {
	Id          string        `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	DataType    v1.Datatype   `json:"datatype"`
	Unit        string        `json:"unit"`
	AccessMode  v1.AccessMode `json:"accessMode"`
	CustomData  *interface{}  `json:"customData,omitempty"`
}

type DataSource struct {
	// id is nil in AgentType, id is NOT nil in Agent
	Id          *string      `json:"id,omitempty"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	DataPoints  []*DataPoint `json:"dataPoints"`
	CustomData  *interface{} `json:"customData,omitempty"`

	DataPointById map[string]*DataPoint `json:"-"`
}

// AgentType is implemented as a template of agent.
// An agent is generated from the agentType, the data sources of the agent is a copy of the agentType.
// In other word, the change of agentType shall not be propagated to the agents generated from the agentType.
type AgentType struct {
	meta.ObjectMeta
	Description *string       `json:"description,omitempty"`
	DataSources []*DataSource `json:"dataSources"`

	DataSourceByName map[string]*DataSource `json:"-"`
}

type DataPointMapping struct {
	meta.ObjectMeta
	AgentId            string        `json:"agentId"`
	DataSourceId       string        `json:"dataSourceId"`
	DataPointId        string        `json:"dataPointId"`
	DataPointUnit      string        `json:"dataPointUnit"`
	DataPointType      v1.Datatype   `json:"dataPointType"`
	ThingId            string        `json:"thingId"`
	PropertySetName    string        `json:"propertySetName"`
	ThingName          string        `json:"thingName"`
	PropertyName       string        `json:"propertyName"`
	PropertyUnit       string        `json:"propertyUnit"`
	PropertyType       v1.Datatype   `json:"propertyType"`
	PropertyAccessMode v1.AccessMode `json:"propertyAccessMode"`
}

type PropertySetInfo struct {
	ThingId         string
	PropertySetName string
}

type PropertyInfo struct {
	PropertySetInfo
	PropertyName string
}

type DataPointInfo struct {
	AgentId      string
	DataSourceId string
	DataPointId  string
}

type ActionData struct {
	Seq       string              `json:"seq" binding:"required"`
	Timestamp int64               `json:"timestamp" binding:"required"`
	Values    []v1.DataPointValue `json:"values" binding:"required,dive"`
}

type CommandData struct {
	Seq       string                 `json:"seq" binding:"required"`
	Timestamp int64                  `json:"timestamp" binding:"required"`
	Type      string                 `json:"type,required"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

type AckData struct {
	Seq       string    `json:"seq" binding:"required"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Code      int       `json:"code,required"`
	Message   *string   `json:"message,omitempty"`
}

type ResponseModel struct {
	PropertySetTypes  interface{} `json:"propertySetTypes,omitempty"`
	ThingTypes        interface{} `json:"thingTypes,omitempty"`
	Things            interface{} `json:"things,omitempty"`
	Characteristics   interface{} `json:"characteristics,omitempty"`
	PropertySets      interface{} `json:"propertySets,omitempty"`
	AgentTypes        interface{} `json:"agentTypes,omitempty"`
	Agents            interface{} `json:"agents,omitempty"`
	DataPointMappings interface{} `json:"dataPointMappings,omitempty"`
}

type EventType int8

const (
	Create EventType = iota
	Update
	Remove
)

type Event struct {
	Type EventType
	Data interface{}
}

func (et EventType) String() string {
	return []string{
		"Create",
		"Update",
		"Delete",
	}[et]
}

type Predicate func(value interface{}) bool
