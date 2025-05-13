package runtime

import (
	html "html/template"
	"lightiot/pkg/generic/meta"
	v1 "lightiot/pkg/notification/v1"
	text "text/template"
)

type RecipientIem struct {
	Address string         `json:"address" binding:"required,min=1,max=64"`
	Type    v1.ChannelType `json:"type" binding:"required"`
}

type Recipient struct {
	meta.ObjectMeta
	Detail []RecipientIem `json:"detail" binding:"required,dive"`
}

type PlaceholderItem struct {
	Name         string      `json:"name"`
	DefaultValue interface{} `json:"defaultValue"`
}

type TemplatePlaceholder struct {
	ETag         string            `json:"eTag"`
	Placeholders []PlaceholderItem `json:"placeholders"`
}

type ParsedTmpl struct {
	TextTmpl *text.Template `json:"-"`
	HtmlTmpl *html.Template `json:"-"`
}

type MessageTemplate struct {
	meta.ObjectMeta
	Type        v1.ContentType       `json:"type"`
	Content     string               `json:"content"`
	ParsedTmpl  ParsedTmpl           `json:"-"`
	Placeholder *TemplatePlaceholder `json:"placeholder,omitempty"`
}

type TmplRecipient struct {
	Id string `json:"id"`
}

type Template struct {
	meta.ObjectMeta
	Subject          string          `json:"subject" binding:"required,min=1,max=77"`
	From             string          `json:"from" binding:"required,min=1,max=64"`
	Recipients       []TmplRecipient `json:"recipients" binding:"required,dive"`
	MessageTemplates []string        `json:"messageTemplates" binding:"required"`
}

type Message struct {
	meta.ObjectMeta
	*v1.Message `json:",inline"`

	MsgTmpls []*MessageTemplate `json:"-"`
}

type Config struct {
	meta.ObjectMeta
	v1.Config
}

// TODO
// type ConfigStore struct {
// 	Tenant string `json:"-"`
// 	Data   []byte `json:"data"`
// }

type IsReferredByTemplateFunc func(id string) bool

type ResponseModel struct {
	ServerConfigurations interface{} `json:"serverConfigurations,omitempty"`
	Recipients           interface{} `json:"recipients,omitempty"`
	MessageTemplates     interface{} `json:"messageTemplates,omitempty"`
	Placeholders         interface{} `json:"placeholders,omitempty"`
	Templates            interface{} `json:"templates,omitempty"`
}
