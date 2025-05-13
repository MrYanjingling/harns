package runtime

import (
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/promql/parser"
	"time"
)

type ActionType byte

const (
	ActionTypeVirtualParameter ActionType = iota
	ActionTypeEvent
	ActionTypeEmail
	ActionTypeWebhook
	ActionTypeWeCom
	ActionTypeWeChat
)

type BaseAction struct {
	Active     bool              `json:"active"`
	Interval   *generic.Duration `json:"interval,omitempty"`
	lastSendAt time.Time
}

type VirtualParameter struct {
	// does NOT need interval
	*BaseAction
	ThingId         string         `json:"thingId"`
	PropertySetName string         `json:"propertySetName"`
	Property        model.Property `json:"property"`
}

type Event struct {
	*BaseAction
	Severity    int    `json:"severity"`
	Description string `json:"description"`
}

type Webhook struct {
	*BaseAction
	Severity    int      `json:"severity"`
	Description string   `json:"description"`
	Addresses   []string `json:"addresses"`
}

type Notify struct {
	*BaseAction
	Severity    int      `json:"severity"`
	Description string   `json:"description"`
	Addresses   []string `json:"addresses"`
}

type Actions struct {
	Actions map[ActionType]Actioner
}

type Evaluation struct {
	Template   string `json:"template"`
	Expression string `json:"expression"`
}

type Rule struct {
	meta.ObjectMeta
	Description *string      `json:"description"`
	ThingId     string       `json:"thingId"`
	RealTime    bool         `json:"realTime"`
	Active      bool         `json:"active"`
	Evaluations []Evaluation `json:"evaluations"`
	Actions     Actions      `json:"actions,omitempty"`
}

type Expression struct {
	RuleId              string
	RuleName            string
	Expr                parser.Expr
	Properties          []string
	Actions             Actions
	MinInterval         time.Duration
	EvaluationTimestamp time.Time
}

type ResponseModel struct {
	Rules interface{} `json:"rules,omitempty"`
}
