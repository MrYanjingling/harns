package runtime

import (
	"lightiot/pkg/generic/runtime"
)

func (in *CommandType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.Description != nil {
		description := *in.Description
		out.Description = &description
	}

	options := make([]*Option, len(in.Options))
	optionByName := make(map[string]*Option)
	for i, option := range in.Options {
		deepCopyOption := option.DeepCopy()
		options[i] = deepCopyOption
		optionByName[option.Name] = deepCopyOption
	}
	for k, v := range in.OptionByName {
		if _, ok := optionByName[k]; !ok {
			optionByName[k] = v.DeepCopy()
		}
	}
	out.Options = options
	out.OptionByName = optionByName

	return &out
}

func (in *Option) DeepCopy() *Option {
	if in == nil {
		return nil
	}

	out := *in

	if in.Description != nil {
		description := *in.Description
		out.Description = &description
	}

	if in.Values != nil {
		out.Values = make([]string, len(in.Values))
		copy(out.Values, in.Values)
	}

	if in.Min != nil {
		min := *in.Min
		out.Min = &min
	}

	if in.Max != nil {
		max := *in.Max
		out.Max = &max
	}

	return &out
}

func (in *ResponseModel) DeepCopyObject() runtime.Object {
	return in
}
