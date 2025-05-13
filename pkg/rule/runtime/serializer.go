package runtime

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

func (as Actions) MarshalJSON() ([]byte, error) {
	type alias struct {
		VirtualParameter interface{} `json:"virtualParameter,omitempty"`
		Event            interface{} `json:"event,omitempty"`
		Email            interface{} `json:"email,omitempty"`
		Webhook          interface{} `json:"webhook,omitempty"`
		WeCom            interface{} `json:"wecom,omitempty"`
		WeChat           interface{} `json:"wechat,omitempty"`
	}
	a := &alias{}
	for k, v := range as.Actions {
		switch k {
		case ActionTypeVirtualParameter:
			a.VirtualParameter = v
		case ActionTypeEvent:
			a.Event = v
		case ActionTypeEmail:
			a.Email = v
		case ActionTypeWebhook:
			a.Webhook = v
		case ActionTypeWeCom:
			a.WeCom = v
		case ActionTypeWeChat:
			a.WeChat = v
		}
	}
	return json.Marshal(a)
}

func (as *Actions) UnmarshalJSON(bytes []byte) error {
	type alias struct {
		VirtualParameter *VirtualParameter `json:"virtualParameter,omitempty"`
		Event            *Event            `json:"event,omitempty"`
		Email            *Notify           `json:"email,omitempty"`
		Webhook          *Webhook          `json:"webhook,omitempty"`
		WeCom            *Notify           `json:"wecom,omitempty"`
		WeChat           *Notify           `json:"wechat,omitempty"`
	}
	var a alias
	if err := json.Unmarshal(bytes, &a); err != nil {
		return err
	}
	as.Actions = make(map[ActionType]Actioner, 0)
	if a.VirtualParameter != nil {
		as.Actions[ActionTypeVirtualParameter] = a.VirtualParameter
	}
	if a.Event != nil {
		as.Actions[ActionTypeEvent] = a.Event
	}
	if a.Email != nil {
		as.Actions[ActionTypeEmail] = a.Email
	}
	if a.Webhook != nil {
		as.Actions[ActionTypeWebhook] = a.Webhook
	}
	if a.WeCom != nil {
		as.Actions[ActionTypeWeCom] = a.WeCom
	}
	if a.WeChat != nil {
		as.Actions[ActionTypeWeChat] = a.WeChat
	}
	return nil
}

func (as *Actions) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	type alias struct {
		VirtualParameter *VirtualParameter
		Event            *Event
		Email            *Notify
		Webhook          *Webhook
		WeCom            *Notify
		WeChat           *Notify
	}
	a := &alias{}
	for k, v := range as.Actions {
		switch k {
		case ActionTypeVirtualParameter:
			a.VirtualParameter = v.(*VirtualParameter)
		case ActionTypeEvent:
			a.Event = v.(*Event)
		case ActionTypeEmail:
			a.Email = v.(*Notify)
		case ActionTypeWebhook:
			a.Webhook = v.(*Webhook)
		case ActionTypeWeCom:
			a.WeCom = v.(*Notify)
		case ActionTypeWeChat:
			a.WeChat = v.(*Notify)
		}
	}
	err := gob.NewEncoder(&buf).Encode(a)
	return buf.Bytes(), err
}

func (as *Actions) UnmarshalBinary(data []byte) error {
	type alias struct {
		VirtualParameter *VirtualParameter
		Event            *Event
		Email            *Notify
		Webhook          *Webhook
		WeCom            *Notify
		WeChat           *Notify
	}
	var a alias
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&a); err != nil {
		return err
	}
	as.Actions = make(map[ActionType]Actioner, 0)
	if a.VirtualParameter != nil {
		as.Actions[ActionTypeVirtualParameter] = a.VirtualParameter
	}
	if a.Event != nil {
		as.Actions[ActionTypeEvent] = a.Event
	}
	if a.Email != nil {
		as.Actions[ActionTypeEmail] = a.Email
	}
	if a.Webhook != nil {
		as.Actions[ActionTypeWebhook] = a.Webhook
	}
	if a.WeCom != nil {
		as.Actions[ActionTypeWeCom] = a.WeCom
	}
	if a.WeChat != nil {
		as.Actions[ActionTypeWeChat] = a.WeChat
	}
	return nil
}
