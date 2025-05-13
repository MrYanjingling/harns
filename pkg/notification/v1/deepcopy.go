package v1

import "lightiot/pkg/generic/runtime"

func (in *Config) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	return in
}

func (in *Message) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	return in
}

func (in *MessageTmpl) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	return in
}

func (in *Recipient) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	return in
}

func (in *Template) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	return in
}
