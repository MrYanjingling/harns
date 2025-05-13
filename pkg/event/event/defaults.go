package event

import (
	"lightiot/pkg/event/runtime"
	v1 "lightiot/pkg/event/v1"
)

func SetDefaults_EventType(obj *v1.EventType, et *runtime.EventType) {
	if obj.TTL == nil {
		et.TTL = 30
	}

	if obj.ParentId == nil || len(*obj.ParentId) == 0 {
		typeId := runtime.BaseEventTypeId
		et.ParentId = &typeId
	}
}