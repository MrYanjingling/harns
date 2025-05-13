package computation

import (
	event "lightiot/pkg/event/v1"
	model "lightiot/pkg/model/runtime"
	notification "lightiot/pkg/notification/v1"
	"time"
)

type Actuator interface {
	SaveOrUpdateVirtualParameter(thingId, psName string, property *model.Property, ts time.Time, v float64) error
	SendEvent(e *event.StandardEvent) error
	SendNotification(msg *notification.Message) error
}
