package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-data-collector/config"
	"lightiot/pkg/client"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/data/realtimecomputation"
	"lightiot/pkg/data/storage"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/logstorage"
	"time"
)

const (
	_defaultPort               = "32200"
	_defaultWait               = 15 * time.Second
	_defaultInfluxDBUrl        = "http://127.0.0.1:8086"
	_defaultInfluxDBToken      = ""
	_defaultMaxInstances       = 4
	_defaultJobDelaySec        = 5
	_defaultCacheRefreshSec    = 180
	_defaultEventMgrUrl        = "http://127.0.0.1:32500"
	_defaultNotificationMgrUrl = "http://127.0.0.1:32660"
)

type Options struct {
	InfluxDBUrl        string          `json:"influxdb-url"`
	InfluxDBToken      string          `json:"influxdb-token"`
	EnableRule         bool            `json:"enable-rule"`
	EnableRollup       bool            `json:"enable-rollup"`
	EventMgrUrl        string          `json:"iot-event-manager-url"`
	NotificationMgrUrl string          `json:"iot-notification-manager-url"`
	MaxInstances       uint16          `json:"max-consumers"`
	JobDelaySec        int64           `json:"job-delay"`
	CacheRefreshSec    uint16          `json:"cache-refresh"`
	Port               string          `json:"port"`
	Wait               time.Duration   `json:"graceful-timeout"`
	Client             *client.Options `json:",inline"`
	baseoptions.BaseOptions
}

func NewDefaultOptions() *Options {
	return &Options{
		InfluxDBUrl:        _defaultInfluxDBUrl,
		InfluxDBToken:      _defaultInfluxDBToken,
		EnableRule:         false,
		EnableRollup:       false,
		EventMgrUrl:        _defaultEventMgrUrl,
		NotificationMgrUrl: _defaultNotificationMgrUrl,
		MaxInstances:       _defaultMaxInstances,
		JobDelaySec:        _defaultJobDelaySec,
		CacheRefreshSec:    _defaultCacheRefreshSec,
		Port:               _defaultPort,
		Wait:               _defaultWait,
		Client:             client.NewDefaultOptions(),
		BaseOptions:        baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.InfluxDBUrl, "influxdb-url", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVar(&o.InfluxDBToken, "influxdb-token", o.InfluxDBToken, "The InfluxDB token")
	fs.BoolVar(&o.EnableRule, "enable-rule", o.EnableRule, "Enable rule engine")
	fs.BoolVar(&o.EnableRollup, "enable-rollup", o.EnableRollup, "Enable rollup and pre-aggregate")
	fs.StringVar(&o.EventMgrUrl, "iot-event-manager-url", o.EventMgrUrl, "The url of iot-event-manager")
	fs.StringVar(&o.NotificationMgrUrl, "iot-notification-manager-url", o.NotificationMgrUrl, "The url of iot-notification-manager")
	fs.Uint16Var(&o.MaxInstances, "max-consumers", o.MaxInstances, "The maximum number instances of iot-consumer. It has to be the same with --max-consumers in iot-consumer. The actual number of instances cannot exceed --max-consumers")
	fs.Int64Var(&o.JobDelaySec, "job-delay", o.JobDelaySec, "The delay seconds to postpone job execution")
	fs.Uint16Var(&o.CacheRefreshSec, "cache-refresh", o.CacheRefreshSec, "The interval in second between refreshing job cache")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-data-collector exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	o.Client.AddFlags(fs)
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	var (
		actuator                        *realtimecomputation.Actuator
		jm                              *job.Manager
		notifyMgrClient, eventMgrClient client.Client
		err                             error
	)

	c := &config.Config{}

	if notifyMgrClient, err = o.Client.CreateHttpClient(o.NotificationMgrUrl); err != nil {
		return c, err
	}

	if eventMgrClient, err = o.Client.CreateHttpClient(o.EventMgrUrl); err != nil {
		return c, err
	}

	if o.EnableRollup {
		if store, err := logstorage.NewInfluxDB(stopCh, o.InfluxDBUrl, o.InfluxDBToken); err != nil {
			return c, err
		} else {
			jm = job.New(stopCh, o.MaxInstances, o.JobDelaySec, true, o.CacheRefreshSec, store)
		}
	}

	s := storage.NewStore(stopCh, o.InfluxDBUrl, o.InfluxDBToken, storage.WithJobManager(jm), storage.WithRuleEvalFunc(func(key string, rawData storage.RawData) { actuator.Evaluate(key, rawData) }))
	rCh := make(chan gruntime.Object)
	rtc := realtimecomputation.NewManager(stopCh, realtimecomputation.WithRuleChan(rCh))
	actuator = realtimecomputation.NewActuator(stopCh, nil, rtc, s, eventMgrClient, notifyMgrClient)

	s.Init()
	rtc.Init()

	c.Store = s
	return c, nil
}
