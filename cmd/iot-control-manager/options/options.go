package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-control-manager/config"
	"lightiot/pkg/client"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/control/action"
	"lightiot/pkg/control/command"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/control/storage"
	"lightiot/pkg/generic"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/model/propertysettype"
	"lightiot/pkg/model/thing"
	fstorage "lightiot/pkg/storage"
	"time"
)

type Options struct {
	BrokerUrl       string          `json:"iot-data-broker-url"`
	AckTimeout      time.Duration   `json:"ack-timeout"`
	InfluxDBUrl     string          `json:"influxdb-url"`
	InfluxDBToken   string          `json:"influxdb-token"`
	TTL             int             `json:"ttl"`
	Port            string          `json:"port"`
	Wait            time.Duration   `json:"graceful-timeout"`
	MaxInstances    uint16          `json:"max-consumers"`
	JobDelaySec     int64           `json:"job-delay"`
	CacheRefreshSec uint16          `json:"cache-refresh"`
	Client          *client.Options `json:",inline"`
	baseoptions.BaseOptions
}

const (
	_defaultBrokerUrl       = "http://127.0.0.1:32400"
	_defaultAckTimeout      = 30 * time.Second
	_defaultInfluxDBUrl     = "http://127.0.0.1:8086"
	_defaultInfluxDBToken   = ""
	_defaultTTL             = 30
	_defaultPort            = "32130"
	_defaultWait            = time.Second * 15
	_defaultMaxInstances    = 4
	_defaultJobDelaySec     = 5
	_defaultCacheRefreshSec = 180
)

func NewDefaultOptions() *Options {
	return &Options{
		BrokerUrl:       _defaultBrokerUrl,
		AckTimeout:      _defaultAckTimeout,
		InfluxDBUrl:     _defaultInfluxDBUrl,
		InfluxDBToken:   _defaultInfluxDBToken,
		TTL:             _defaultTTL,
		Port:            _defaultPort,
		Wait:            _defaultWait,
		MaxInstances:    _defaultMaxInstances,
		JobDelaySec:     _defaultJobDelaySec,
		CacheRefreshSec: _defaultCacheRefreshSec,
		Client:          client.NewDefaultOptions(),
		BaseOptions:     baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.BrokerUrl, "iot-data-broker-url", o.BrokerUrl, "The url of iot-data-broker")
	fs.DurationVar(&o.AckTimeout, "ack-timeout", o.AckTimeout, "Wait ack of command or action timeout")
	fs.StringVar(&o.InfluxDBUrl, "influxdb-url", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVar(&o.InfluxDBToken, "influxdb-token", o.InfluxDBToken, "The InfluxDB token")
	fs.IntVar(&o.TTL, "ttl", o.TTL, "TTL of command/action history. Unit is Day")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-control-manager exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.Uint16Var(&o.MaxInstances, "max-consumers", o.MaxInstances, "The maximum number instances of iot-consumer. It has to be the same with --max-consumers in iot-consumer. The actual number of instances cannot exceed --max-consumers")
	fs.Int64Var(&o.JobDelaySec, "job-delay", o.JobDelaySec, "The delay seconds to postpone job execution")
	fs.Uint16Var(&o.CacheRefreshSec, "cache-refresh", o.CacheRefreshSec, "The interval in second between refreshing job cache")
	o.Client.AddFlags(fs)
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{
		Config: &runtime.Config{},
	}
	if cli, err := o.Client.CreateHttpClient(o.BrokerUrl); err != nil {
		return c, err
	} else {
		c.IotBrokerClient = cli
	}

	if store, err := logstorage.NewInfluxDB(stopCh, o.InfluxDBUrl, o.InfluxDBToken); err != nil {
		return c, err
	} else {
		c.LogStorage = store
	}

	c.Config.AckTimeout = o.AckTimeout
	c.TTL = o.TTL

	var cm *command.Manager
	pstCh := make(chan gruntime.Object)
	ttCh := make(chan gruntime.Object)
	tCh := make(chan gruntime.Object)
	pstm := propertysettype.NewManager(stopCh, pstCh)
	pstm.InitWatch()
	modelMgr := thing.NewManager(stopCh,
		thing.WithThingTypeChan(ttCh),
		thing.WithThingChan(tCh),
		thing.WithPropertySetTypeManager(pstm),
		thing.WithDeleteThingTypeFunc(func(thingTypeId string) {
			_ = cm.DeleteCommandTypes(thingTypeId)
		}))
	modelMgr.InitWatch()

	ctCh := make(chan gruntime.Object)
	ctStore, _ := generic.NewStore(gruntime.GroupResource{Group: fstorage.StoreGroupToString[fstorage.StoreGroupControl], Resource: fstorage.CommandTypes}, &runtime.CommandType{}, ctCh)
	store := storage.NewStore(stopCh, c.LogStorage, c.TTL)
	am := action.NewManager(stopCh, store, c.IotBrokerClient, modelMgr, c.Config)
	jm := job.New(stopCh, o.MaxInstances, o.JobDelaySec, true, o.CacheRefreshSec, c.LogStorage)
	cm = command.NewManager(stopCh, store, ctStore, ctCh, c.IotBrokerClient, modelMgr, c.Config, jm)
	cm.Init()

	c.ControlConfig.CMDMgr = cm
	c.ControlConfig.ACTMgr = am

	return c, nil
}
