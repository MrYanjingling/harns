package runtime

import "k8s.io/apimachinery/pkg/util/sets"

const (
	BaseEventTypeId = "main.BaseEvent"

	BaseEventFieldId            = "id"
	BaseEventFieldTypeId        = "typeId"
	BaseEventFieldCorrelationId = "correlationId"
	BaseEventFieldTime          = "_time"
	BaseEventFieldThingId       = "thingId"
	BaseEventFieldETag          = "eTag"
	BaseEventFieldFireTime      = "_fire"

	StandardEventTypeId = "main.StandardEvent"

	StandardEventFieldSeverity     = "severity"
	StandardEventFieldDescription  = "description"
	StandardEventFieldCode         = "code"
	StandardEventFieldSource       = "source"
	StandardEventFieldAcknowledged = "acknowledged"
)

var BaseEventFields = sets.NewString(
	BaseEventFieldId,
	BaseEventFieldTypeId,
	BaseEventFieldCorrelationId,
	BaseEventFieldTime,
	BaseEventFieldThingId,
	BaseEventFieldETag,
)

var StandardEventFields = sets.NewString(
	BaseEventFieldId,
	BaseEventFieldTypeId,
	BaseEventFieldCorrelationId,
	BaseEventFieldTime,
	BaseEventFieldThingId,
	BaseEventFieldETag,

	StandardEventFieldSeverity,
	StandardEventFieldDescription,
	StandardEventFieldCode,
	StandardEventFieldSource,
	StandardEventFieldAcknowledged,
)

var StandardEventSeverity = map[int]string{
	20: "Critical",
	30: "Error",
	40: "Warning",
	50: "Information",
}
