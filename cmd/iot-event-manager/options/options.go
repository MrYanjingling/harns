package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-event-manager/config"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/event/event"
	"lightiot/pkg/event/storage"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/logstorage"
	"time"
)

type Options struct {
	InfluxDBUrl     string        `json:"influxdb-url"`
	InfluxDBToken   string        `json:"influxdb-token"`
	Port            string        `json:"port"`
	Wait            time.Duration `json:"graceful-timeout"`
	MaxInstances    uint16        `json:"max-consumers"`
	JobDelaySec     int64         `json:"job-delay"`
	CacheRefreshSec uint16        `json:"cache-refresh"`
	baseoptions.BaseOptions
}

const (
	_defaultInfluxDBUrl     = "http://127.0.0.1:8086"
	_defaultInfluxDBToken   = ""
	_defaultPort            = "32500"
	_defaultWait            = 15 * time.Second
	_defaultMaxInstances    = 4
	_defaultJobDelaySec     = 5
	_defaultCacheRefreshSec = 180
)

func NewDefaultOptions() *Options {
	return &Options{
		InfluxDBUrl:     _defaultInfluxDBUrl,
		InfluxDBToken:   _defaultInfluxDBToken,
		Port:            _defaultPort,
		Wait:            _defaultWait,
		MaxInstances:    _defaultMaxInstances,
		JobDelaySec:     _defaultJobDelaySec,
		CacheRefreshSec: _defaultCacheRefreshSec,
		BaseOptions:     baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.InfluxDBUrl, "influxdb-url", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVar(&o.InfluxDBToken, "influxdb-token", o.InfluxDBToken, "The InfluxDB token")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-event-manager exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.Uint16Var(&o.MaxInstances, "max-consumers", o.MaxInstances, "The maximum number instances of iot-consumer. It has to be the same with --max-consumers in iot-consumer. The actual number of instances cannot exceed --max-consumers")
	fs.Int64Var(&o.JobDelaySec, "job-delay", o.JobDelaySec, "The delay seconds to postpone job execution")
	fs.Uint16Var(&o.CacheRefreshSec, "cache-refresh", o.CacheRefreshSec, "The interval in second between refreshing job cache")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{}

	if store, err := logstorage.NewInfluxDB(stopCh, o.InfluxDBUrl, o.InfluxDBToken); err != nil {
		return c, err
	} else {
		c.LogStorage = store
	}

	etCh := make(chan gruntime.Object)
	tCh := make(chan gruntime.Object)
	aetCh := make(chan gruntime.Object, 1024) // active thing channel
	s := storage.NewStore(stopCh, c.LogStorage)
	ts, _ := storage.NewTypeStore(etCh, tCh, aetCh)
	jm := job.New(stopCh, o.MaxInstances, o.JobDelaySec, true, o.CacheRefreshSec, s.GetLogStorage())
	m := event.NewManager(stopCh, s, ts, etCh, jm, tCh)

	m.Init()

	c.EventConfig.EventMgr = m

	return c, nil
}
