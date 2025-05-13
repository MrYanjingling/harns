package v1

import (
	gruntime "lightiot/pkg/generic/runtime"
)

func (in *CommandType) DeepCopyObject() gruntime.Object {
	return in
}

