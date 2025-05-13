package options

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	flag "github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"lightiot/cmd/iot-data-broker/config"
	"lightiot/pkg/client"
	"lightiot/pkg/exchange"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/agent"
	"lightiot/pkg/model/datapointmapping"
	"lightiot/pkg/model/runtime"
	"os"
	"time"
)

const (
	_defaultMqttUsername = ""
	_defaultMqttPassword = ""
	_defaultQueueLength  = 1024
	_defaultCollectorUrl = "http://127.0.0.1:32200"
	_defaultEventMgrUrl  = "http://127.0.0.1:32500"
	_defaultDaemon       = false
	_defaultBiDirection  = false
	_defaultAckTimeout   = 20 * time.Second
	_defaultPort         = "32400"
	_defaultWait         = time.Second * 15
)

var (
	_defaultMqttBrokerUrls = []string{"tcp://127.0.0.1:1883"}
)

type Options struct {
	MqttBrokerUrls []string        `json:"mqtt-broker-urls"`
	MqttUsername   string          `json:"mqtt-username"`
	MqttPassword   string          `json:"mqtt-password"`
	QueueLength    int             `json:"queue-length"`
	CollectorUrl   string          `json:"iot-data-collector-url"`
	EventMgrUrl    string          `json:"iot-event-manager-url"`
	Daemon         bool            `json:"daemon"`
	BiDirection    bool            `json:"bi-direction"`
	AckTimeout     time.Duration   `json:"ack-timeout"`
	Port           string          `json:"port"`
	Wait           time.Duration   `json:"graceful-timeout"`
	Client         *client.Options `json:",inline"`
	baseoptions.BaseOptions
}

func NewDefaultOptions() *Options {
	return &Options{
		MqttBrokerUrls: _defaultMqttBrokerUrls,
		MqttUsername:   _defaultMqttUsername,
		MqttPassword:   _defaultMqttPassword,
		QueueLength:    _defaultQueueLength,
		CollectorUrl:   _defaultCollectorUrl,
		EventMgrUrl:    _defaultEventMgrUrl,
		Daemon:         _defaultDaemon,
		BiDirection:    _defaultBiDirection,
		AckTimeout:     _defaultAckTimeout,
		Port:           _defaultPort,
		Wait:           _defaultWait,
		Client:         client.NewDefaultOptions(),
		BaseOptions:    baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringSliceVarP(&o.MqttBrokerUrls, "mqtt-broker-urls", "", o.MqttBrokerUrls, "The MQTT broker urls. The format should be scheme://host:port Where \"scheme\" is one of \"tcp\", \"ssl\", or \"ws\"")
	fs.StringVarP(&o.MqttUsername, "mqtt-username", "u", o.MqttUsername, "The MQTT username")
	fs.StringVarP(&o.MqttPassword, "mqtt-password", "p", o.MqttPassword, "The MQTT password")
	fs.IntVarP(&o.QueueLength, "queue-length", "", o.QueueLength, "The queue length of mqtt topic receiver")
	fs.StringVarP(&o.CollectorUrl, "iot-data-collector-url", "", o.CollectorUrl, "The url of iot-data-collector")
	fs.StringVarP(&o.EventMgrUrl, "iot-event-manager-url", "", o.EventMgrUrl, "The url of iot-event-manager")
	fs.BoolVarP(&o.Daemon, "daemon", "d", o.Daemon, "Whether iot data broker runs as a daemon")
	fs.BoolVarP(&o.BiDirection, "bi-direction", "", o.BiDirection, "Whether broker dispatches control action/command to agent")
	fs.DurationVarP(&o.AckTimeout, "ack-timeout", "", o.AckTimeout, "Wait ack of command or action timeout")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-data-broker exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	o.Client.AddFlags(fs)
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	// init mqtt client
	mqttOption := mqtt.NewClientOptions()
	for _, s := range o.MqttBrokerUrls {
		mqttOption = mqttOption.AddBroker(s)
	}
	mqttOption.SetUsername(o.MqttUsername)
	mqttOption.SetPassword(o.MqttPassword)
	mqttOption.SetOrderMatters(false)
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		// TODO set Pod Name
		mqttOption.SetClientID("<Pod-Name>")
	} else {
		// TODO each instance should have a unique client ID
		mqttOption.SetClientID("iot-data-broker-" + o.Port)
	}
	mqttClient := mqtt.NewClient(mqttOption)
	klog.V(1).InfoS("Connected to MQTT", "servers", o.MqttBrokerUrls)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		klog.ErrorS(token.Error(), "Failed to connect MQTT", "servers", o.MqttBrokerUrls)
		return nil, token.Error()
	}

	var (
		collectorClient, eventMgrClient client.Client
		err                             error
	)
	if collectorClient, err = o.Client.CreateHttpClient(o.CollectorUrl); err != nil {
		return nil, err
	}
	if eventMgrClient, err = o.Client.CreateHttpClient(o.EventMgrUrl); err != nil {
		return nil, err
	}

	em := exchange.NewManager(stopCh, mqttClient, o.QueueLength, collectorClient, eventMgrClient, o.BiDirection, o.AckTimeout)

	aCh := make(chan gruntime.Object)
	mCh := make(chan gruntime.Object)
	am := agent.NewManager(stopCh, agent.WithAgentChan(aCh), agent.WithExchangeManager(em))
	mm := datapointmapping.NewManager(stopCh, datapointmapping.WithMappingChan(mCh), datapointmapping.WithExchangeManager(em))

	// BE CARE to change, the following order is important
	for _, m := range mm.LoadMappings() {
		exchange.OnDataPointMappingReceived(m, em, runtime.Create)
	}
	mm.InitWatch(exchange.OnDataPointMappingReceived)
	for _, a := range am.LoadAgents() {
		exchange.OnAgentReceived(a, nil, em, runtime.Create)
	}
	am.InitWatch(exchange.OnAgentReceived)

	return &config.Config{ExchangeMgr: em}, nil
}
