package notification

import (
	"lightiot/pkg/generic/endpoints"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/config"
	"lightiot/pkg/notification/message"
	"lightiot/pkg/notification/messagetemplate"
	"lightiot/pkg/notification/recipient"
	"lightiot/pkg/notification/template"
)

// TODO how to resolve the conflicts between 202 and 201 after creating success
// func createMessage(m *message.Manager) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		obj := &v1.Message{}
// 		if err := c.ShouldBindJSON(obj); err != nil {
// 			klog.V(3).InfoS("Failed to parse message", "err", err)
// 			c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
// 			return
// 		}
// 		err := m.SendMessage(obj)
//
// 		if err != nil {
// 			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
// 			return
// 		}
//
// 		// TODO use different scheme
// 		c.Status(http.StatusAccepted)
// 	}
// }

type Config struct {
	Config    *config.REST
	Recipient *recipient.REST
	MsgTmpl   *messagetemplate.REST
	PHTmpl    *messagetemplate.PlaceHolderREST
	Tmpl      *template.REST
	Msg       *message.REST
}

type RESTProvider struct{}

func (p RESTProvider) NewREST(methods []string, config interface{}) (endpoints.APIGroupVersion, error) {
	c, _ := config.(*Config)
	mgr := map[string]interface{}{
		"recipients":                    c.Recipient,
		"messageTemplates":              c.MsgTmpl,
		"messageTemplates/placeholders": c.PHTmpl,
		"templates":                     c.Tmpl,
		"messages":                      c.Msg,
		"serverConfigurations":          c.Config,
	}
	gv := endpoints.APIGroupVersion{
		Managers:     mgr,
		Methods:      methods,
		GroupVersion: gruntime.GroupVersion{"notification", "v1"},
	}
	return gv, nil
}
