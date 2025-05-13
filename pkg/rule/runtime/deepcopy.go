package runtime

import (
	"lightiot/pkg/generic/runtime"
)

func (in *Rule) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in

	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}

	if in.Evaluations != nil {
		out.Evaluations = make([]Evaluation, len(in.Evaluations))
		copy(out.Evaluations, in.Evaluations)
	}

	out.Actions = *in.Actions.DeepCopy()
	return &out
}

func (in *Actions) DeepCopy() *Actions {
	if in == nil {
		return nil
	}
	out := *in

	if in.Actions != nil {
		out.Actions = make(map[ActionType]Actioner, len(in.Actions))
		for k, v := range in.Actions {
			out.Actions[k] = v.DeepCopy()
		}
	}
	return &out
}

func (in *Webhook) DeepCopy() Actioner {
	if in == nil {
		return nil
	}
	out := *in

	if in.Addresses != nil {
		out.Addresses = make([]string, len(in.Addresses))
		copy(out.Addresses, in.Addresses)
	}

	out.BaseAction = in.BaseAction.DeepCopy()
	return &out
}

func (in *Notify) DeepCopy() Actioner {
	if in == nil {
		return nil
	}
	out := *in

	if in.Addresses != nil {
		out.Addresses = make([]string, len(in.Addresses))
		copy(out.Addresses, in.Addresses)
	}

	out.BaseAction = in.BaseAction.DeepCopy()
	return &out
}

func (in *Event) DeepCopy() Actioner {
	if in == nil {
		return nil
	}
	out := *in
	out.BaseAction = in.BaseAction.DeepCopy()
	return &out
}

func (in *VirtualParameter) DeepCopy() Actioner {
	if in == nil {
		return nil
	}
	out := *in
	out.BaseAction = in.BaseAction.DeepCopy()
	return &out
}

func (in *BaseAction) DeepCopy() *BaseAction {
	if in == nil {
		return nil
	}

	out := *in
	return &out
}

func (in *ResponseModel) DeepCopyObject() runtime.Object {
	return in
}
