package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-notification-manager/config"
	"lightiot/pkg/client"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	configServer "lightiot/pkg/notification/config"
	"lightiot/pkg/notification/message"
	"lightiot/pkg/notification/messagetemplate"
	"lightiot/pkg/notification/recipient"
	"lightiot/pkg/notification/template"
	"time"
)

const (
	_defaultWeComApiUrl  = "https://qyapi.weixin.qq.com"
	_defaultWeChatApiUrl = "https://api.weixin.qq.com"
	_defaultPort         = "32660"
	_defaultWait         = time.Second * 15
)

type Options struct {
	WeComApiUrl  string          `json:"wecom-api-url"`
	WeChatApiUrl string          `json:"wechat-api-url"`
	Port         string          `json:"port"`
	Wait         time.Duration   `json:"graceful-timeout"`
	Client       *client.Options `json:",inline"`
	baseoptions.BaseOptions
}

func NewDefaultOptions() *Options {
	return &Options{
		WeComApiUrl:  _defaultWeComApiUrl,
		WeChatApiUrl: _defaultWeChatApiUrl,
		Port:         _defaultPort,
		Wait:         _defaultWait,
		Client:       client.NewDefaultOptions(),
		BaseOptions:  baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVarP(&o.WeComApiUrl, "wecom-api-url", "", o.WeComApiUrl, "WeCom API URL")
	fs.StringVarP(&o.WeChatApiUrl, "wechat-api-url", "", o.WeChatApiUrl, "WeChat API URL")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-notification-manager exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	o.Client.AddFlags(fs)
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{}

	weComClient, err := o.Client.CreateHttpClient(o.WeComApiUrl)
	if err != nil {
		return c, err
	}

	weChatClient, err := o.Client.CreateHttpClient(o.WeChatApiUrl)
	if err != nil {
		return c, err
	}

	rCh := make(chan gruntime.Object)
	mtCh := make(chan gruntime.Object)
	tCh := make(chan gruntime.Object)
	mCh := make(chan gruntime.Object)
	cCh := make(chan gruntime.Object)

	configMgr := configServer.NewManager(stopCh, cCh)
	var tmplMgr *template.Manager
	recipientMgr := recipient.NewManager(stopCh, rCh, configMgr,
		recipient.WithUpdateTemplateFunc(func(recipientID string) error { return tmplMgr.UpdateTemplateByRecipientDeletion(recipientID) }),
		recipient.WithForbidDeletionFunc(func(recipientID string) bool { return tmplMgr.IsRecipientReferredByTemplate(recipientID) }),
	)
	msgTmplMgr := messagetemplate.NewManager(stopCh, mtCh,
		messagetemplate.WithUpdateTemplateFunc(func(mtId string) error { return tmplMgr.UpdateTemplateByMessageTmplDeletion(mtId) }),
		messagetemplate.WithForbidDeletionFunc(func(mtId string) bool { return tmplMgr.IsMessageTemplateReferredByTemplate(mtId) }),
	)
	tmplMgr = template.NewManager(stopCh, tCh, recipientMgr, msgTmplMgr)
	msgMgr := message.NewManager(stopCh, mCh, configMgr, recipientMgr, msgTmplMgr, tmplMgr, weChatClient, weComClient)

	msgMgr.Init()
	configMgr.Init()
	recipientMgr.Init()
	msgTmplMgr.Init()
	tmplMgr.Init()

	c.NotifyConfig.Config = configServer.NewREST(configMgr)
	c.NotifyConfig.Recipient = recipient.NewREST(recipientMgr)
	c.NotifyConfig.MsgTmpl = messagetemplate.NewREST(msgTmplMgr)
	c.NotifyConfig.PHTmpl = messagetemplate.NewRESTPlaceHolder(msgTmplMgr)
	c.NotifyConfig.Tmpl = template.NewREST(tmplMgr)
	c.NotifyConfig.Msg = message.NewREST(msgMgr)

	return c, nil
}
