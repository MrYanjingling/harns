package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-rule-manager/config"
	"lightiot/pkg/client"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/rule/rule"
	"time"
)

type Options struct {
	ModelMgrUrl        string          `json:"iot-model-manager-url"`
	NotificationMgrUrl string          `json:"iot-notification-manager-url"`
	Port               string          `json:"port"`
	Wait               time.Duration   `json:"graceful-timeout"`
	Client             *client.Options `json:",inline"`
	baseoptions.BaseOptions
}

const (
	_defaultPort               = "32600"
	_defaultWait               = 15 * time.Second
	_defaultModelMgrUrl        = "http://127.0.0.1:32100"
	_defaultNotificationMgrUrl = "http://127.0.0.1:32660"
)

func NewDefaultOptions() *Options {
	return &Options{
		Port:               _defaultPort,
		Wait:               _defaultWait,
		ModelMgrUrl:        _defaultModelMgrUrl,
		NotificationMgrUrl: _defaultNotificationMgrUrl,
		Client:             client.NewDefaultOptions(),
		BaseOptions:        baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVarP(&o.ModelMgrUrl, "iot-model-manager-url", "", o.ModelMgrUrl, "The url of iot-model-manager")
	fs.StringVarP(&o.NotificationMgrUrl, "iot-notification-manager-url", "", o.NotificationMgrUrl, "The url of iot-notification-manager")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-rule-manager exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	o.Client.AddFlags(fs)
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{}
	rCh := make(chan gruntime.Object)
	tCh := make(chan gruntime.Object)
	ttCh := make(chan gruntime.Object)

	modelClient, err := o.Client.CreateHttpClient(o.ModelMgrUrl)
	if err != nil {
		return c, err
	}
	notificationClient, err := o.Client.CreateHttpClient(o.NotificationMgrUrl)
	if err != nil {
		return c, err
	}

	m := rule.NewManager(stopCh, rule.WithRuleChan(rCh), rule.WithModelClient(modelClient), rule.WithNotificationClient(notificationClient), rule.WithThingChan(tCh), rule.WithThingTypeChan(ttCh))
	m.Init()

	c.RuleConfig.RuleMgr = m

	return c, nil
}
