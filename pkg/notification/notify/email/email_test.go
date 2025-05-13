package email

import (
	"bytes"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"testing"
	"text/template"
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

func TestEmail(t *testing.T) {
	rm := &runtime.Message{
		ObjectMeta: meta.ObjectMeta{},
		Message: &v1.Message{
			Channel: 0,
			Subject: "",
			From:    "",
			Payload: map[string]interface{}{
				"content": "峨眉山会议室温度超过 30 摄氏度。",
			},
			To:         nil,
			TemplateId: "",
		},
		MsgTmpls: []*runtime.MessageTemplate{simpleMessageTmpl},
	}

	var buf bytes.Buffer
	err := rm.MsgTmpls[0].ParsedTmpl.TextTmpl.Execute(&buf, rm)
	if err != nil {
		t.Error(err)
	}
	t.Log(buf.String())
}
