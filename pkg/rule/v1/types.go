package v1

import (
	"lightiot/pkg/generic"
	model "lightiot/pkg/model/runtime"
)

type BaseAction struct {
	Active   bool              `json:"active"`
	Interval *generic.Duration `json:"interval"`
}

func (ba *BaseAction) IsActive() bool {
	return ba != nil && ba.Active
}

type VirtualParameterAction struct {
	// does NOT need interval
	*BaseAction
	ThingId         string         `json:"thingId"`
	PropertySetName string         `json:"propertySetName"`
	Property        model.Property `json:"property"`
}

type EventAction struct {
	*BaseAction
	Severity    int    `json:"severity" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type NotifyAction struct {
	*BaseAction
	Description string   `json:"description" binding:"required"`
	Addresses   []string `json:"addresses" binding:"required,min=1"`
	Severity    int      `json:"severity"`
}

type Action struct {
	VirtualParameter *VirtualParameterAction `json:"virtualParameter,omitempty"`
	Event            *EventAction            `json:"event,omitempty"`
	Email            *NotifyAction           `json:"email,omitempty"`
	Webhook          *NotifyAction           `json:"webhook,omitempty"`
	WeCom            *NotifyAction           `json:"wecom,omitempty"`
	WeChat           *NotifyAction           `json:"wechat,omitempty"`
}

type Evaluation struct {
	Template   string `json:"template"`
	Expression string `json:"expression"`
}

type Rule struct {
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	ThingId     string       `json:"thingId"`
	RealTime    bool         `json:"realTime"`
	Active      bool         `json:"active"`
	Evaluations []Evaluation `json:"evaluations"`
	Actions     Action       `json:"actions,omitempty"`
}
