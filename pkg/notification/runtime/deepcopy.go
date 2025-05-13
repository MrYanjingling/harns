package runtime

import (
	gruntime "lightiot/pkg/generic/runtime"
	v1 "lightiot/pkg/notification/v1"
)

func (in *Config) DeepCopyObject() gruntime.Object {
	if in == nil {
		return nil
	}
	var out Config
	out = *in
	if in.Email != nil {
		out.Email = new(v1.EmailConfig)
		out.Email.Smtp = in.Email.Smtp
	}
	if in.WeCom != nil {
		out.WeCom = new(v1.WeComConfig)
		*out.WeCom = *in.WeCom
	}
	if in.WeChat != nil {
		out.WeChat = new(v1.WeChatConfig)
		*out.WeChat = *in.WeChat
	}
	return &out
}

func (in *TemplatePlaceholder) DeepCopyObject() gruntime.Object {
	if in == nil {
		return nil
	}
	out := *in

	if in.Placeholders != nil {
		out.Placeholders = make([]PlaceholderItem, len(in.Placeholders))
		copy(out.Placeholders, in.Placeholders)
	}
	return &out
}

func (in *MessageTemplate) DeepCopyObject() gruntime.Object {
	if in == nil {
		return nil
	}
	out := *in
	if in.Placeholder != nil {
		out.Placeholder = in.Placeholder.DeepCopyObject().(*TemplatePlaceholder)
	}
	return in
}

func (in *Recipient) DeepCopyObject() gruntime.Object {
	if in == nil {
		return nil
	}
	return in
}

func (in *Template) DeepCopyObject() gruntime.Object {
	if in == nil {
		return nil
	}
	return in
}

func (in *ResponseModel) DeepCopyObject() gruntime.Object {
	return in
}
