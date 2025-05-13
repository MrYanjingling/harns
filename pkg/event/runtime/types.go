package runtime

import (
	"k8s.io/apimachinery/pkg/util/sets"
	v1 "lightiot/pkg/event/v1"
	"lightiot/pkg/generic/meta"
)

type EventType struct {
	meta.ObjectMeta
	ParentId *string  `json:"parentId,omitempty"`
	TTL      int      `json:"ttl"`
	Fields   []*Field `json:"fields"`

	FieldByName map[string]*Field `json:"-"`
	Parents     []*EventType      `json:"parents,omitempty"`
	RequiredCnt int               `json:"-"`
}

type Field struct {
	Name       string      `json:"name"`
	Filterable bool        `json:"filterable"`
	Required   bool        `json:"required"`
	Updatable  bool        `json:"updatable"`
	DataType   v1.Datatype `json:"datatype"`
	Values     interface{} `json:"values"`
}

type BaseEvent struct {
	Id            string `json:"id"`
	TypeId        string `json:"typeId"`
	CorrelationId string `json:"correlationId"`
	Time          string `json:"_time"`
	ThingId       string `json:"thingId"`
	ETag          string `json:"eTag"`
}

type StandardEvent struct {
	BaseEvent    `json:",inline"`
	Severity     int    `json:"severity,omitempty"`
	Description  string `json:"description,omitempty"`
	Code         string `json:"code,omitempty"`
	Source       string `json:"source,omitempty"`
	Acknowledged bool   `json:"acknowledged,omitempty"`
}

type ResponseModel struct {
	EventTypes interface{} `json:"eventTypes,omitempty"`
	Events     interface{} `json:"events,omitempty"`
}

type ActiveEventType struct {
	meta.ObjectMeta
	ThingId    string
	EventTypes sets.String
}
