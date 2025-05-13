package runtime

import (
	"k8s.io/apimachinery/pkg/util/sets"
	v1 "lightiot/pkg/event/v1"
	"lightiot/pkg/generic/runtime"
)

func (in *EventType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in

	out.Fields = make([]*Field, len(in.Fields))
	out.FieldByName = make(map[string]*Field, len(in.FieldByName))

	for i, field := range in.Fields {
		out.Fields[i] = field.DeepCopy()
		out.FieldByName[out.Fields[i].Name] = out.Fields[i]
	}

	for key, field := range in.FieldByName {
		if _, ok := out.FieldByName[key]; !ok {
			out.FieldByName[key] = field
		}
	}
	return &out
}

func (in *Field) DeepCopy() *Field {
	if in == nil {
		return nil
	}

	out := *in
	switch in.DataType {
	case v1.DatatypeEnum:
		inValues := in.Values.([]interface{})
		out.Values = make([]interface{}, len(inValues))
		copy(out.Values.([]interface{}), inValues)
	case v1.DatatypeMap:
		inValues := in.Values.(map[string]interface{})
		values := make(map[string]interface{}, len(inValues))
		for k, v := range inValues {
			values[k] = v
		}
		out.Values = values
	default:
		out.Values = nil
	}

	return &out
}

func (in *ResponseModel) DeepCopyObject() runtime.Object {
	return in
}

func (in *ActiveEventType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.EventTypes.Len() != 0 {
		out.EventTypes = sets.NewString(in.EventTypes.UnsortedList()...)
	}

	return &out
}
