package v1

import "lightiot/pkg/generic/runtime"

func (in *EventType) DeepCopyObject() runtime.Object {
	return in
}

func (in *Field) DeepCopy() *Field {
	return in
}
