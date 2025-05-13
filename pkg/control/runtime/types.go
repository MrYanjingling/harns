package runtime

import (
	v1 "lightiot/pkg/control/v1"
	"lightiot/pkg/generic/meta"
	"time"
)

type Option struct {
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Filterable  bool        `json:"filterable"`
	Datatype    v1.Datatype `json:"datatype"`
	Values      []string    `json:"values,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
}

type CommandType struct {
	meta.ObjectMeta
	TypeId       string             `json:"-"`
	Description  *string            `json:"description,omitempty"`
	Ack          bool               `json:"ack"`
	User         string             `json:"-"`
	ThingTypeId  string             `json:"thingTypeId"`
	Options      []*Option          `json:"options,omitempty"`
	OptionByName map[string]*Option `json:"-"`
}

type State byte

const (
	StateDelivering State = iota
	StateDelivered
	StateSuccess
	StateFail
)

var stateToString = map[State]string{
	StateDelivering: "delivering",
	StateDelivered:  "delivered",
	StateSuccess:    "success",
	StateFail:       "fail",
}

var stateFromString = map[string]State{
	"delivering": StateDelivering,
	"delivered":  StateDelivered,
	"success":    StateSuccess,
	"fail":       StateFail,
}

func (ste State) String() string {
	return stateToString[ste]
}

type Command struct {
	Seq     int64                  `json:"seq"`
	ThingId string                 `json:"thingId"`
	TypeId  string                 `json:"typeId"`
	Time    time.Time              `json:"_time"`
	State   State                  `json:"state"`
	Ack     bool                   `json:"ack,omitempty"`
	Timeout time.Duration          `json:"timeout,omitempty"`
	Code    int                    `json:"code,omitempty"`
	Message *string                `json:"message,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type Action struct {
	Seq     int64                  `json:"seq"`
	ThingId string                 `json:"thingId"`
	PsName  string                 `json:"PropertySetName"`
	Time    time.Time              `json:"_time"`
	State   State                  `json:"state"`
	Ack     bool                   `json:"ack,omitempty"`
	Timeout time.Duration          `json:"timeout,omitempty"`
	Code    int                    `json:"code,omitempty"`
	Message *string                `json:"message,omitempty"`
	Actions map[string]interface{} `json:"options,omitempty"`
}

type ResponseModel struct {
	CommandTypes interface{} `json:"commandTypes,omitempty"`
	Commands     interface{} `json:"commands,omitempty"`
	Actions      interface{} `json:"actions,omitempty"`
}

// Config info

type Config struct {
	AckTimeout time.Duration
}
