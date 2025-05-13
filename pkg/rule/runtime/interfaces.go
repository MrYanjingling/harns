package runtime

import (
	notification "lightiot/pkg/notification/v1"
	"lightiot/pkg/rule/computation"
	"time"
)

type BaseActioner interface {
	IsActive() bool
	GetInterval() time.Duration
	SetLastSendAt(ts time.Time)
	GetLastSendAt() time.Time
}

type Sender interface {
	Send(ct notification.ChannelType, thingId, ruleName string, ts time.Time, actualValue float64, actuator computation.Actuator) error
}

type DeepCopy interface {
	DeepCopy() Actioner
}

type Actioner interface {
	BaseActioner
	Sender
	DeepCopy
}
