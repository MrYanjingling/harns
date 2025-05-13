package message

import (
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"text/template"
)

const (
	simpleMessagePayloadFieldContent = "content"
)

var (
	tmpl, _           = template.New("simpleMsg").Parse(`{{.Payload.content}}`)
	simpleMessageTmpl = &runtime.MessageTemplate{
		Type: v1.ContentTypeText,
		ParsedTmpl: runtime.ParsedTmpl{
			TextTmpl: tmpl,
		},
	}
)
