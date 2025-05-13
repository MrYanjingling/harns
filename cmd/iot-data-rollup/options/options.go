package options

import (
	"lightiot/cmd/iot-data-rollup/config"
	baseoptions "lightiot/pkg/generic/options"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/model/propertysettype"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	"lightiot/pkg/rollup/data"
	"time"

	flag "github.com/spf13/pflag"
)

const (
	_defaultInfluxDBUrl   = "http://127.0.0.1:8086"
	_defaultInfluxDBToken = ""
	_defaultPort          = "32320"
	_defaultWait          = time.Second * 15
)

type Options struct {
	InfluxDBUrl   string        `json:"influxdb-url"`
	InfluxDBToken string        `json:"influxdb-token"`
	Port          string        `json:"port"`
	Wait          time.Duration `json:"graceful-timeout"`
	baseoptions.BaseOptions
}

func NewDefaultOptions() *Options {
	return &Options{
		InfluxDBUrl:   _defaultInfluxDBUrl,
		InfluxDBToken: _defaultInfluxDBToken,
		Port:          _defaultPort,
		Wait:          _defaultWait,
		BaseOptions:   baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.InfluxDBUrl, "influxdb-url", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVar(&o.InfluxDBToken, "influxdb-token", o.InfluxDBToken, "The InfluxDB token")
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-data-rollup exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	store, err := logstorage.NewInfluxDB(stopCh, o.InfluxDBUrl, o.InfluxDBToken)
	if err != nil {
		return nil, err
	}

	var rollupManager *data.Manager
	pstCh := make(chan gruntime.Object)
	ttCh := make(chan gruntime.Object)
	tCh := make(chan gruntime.Object)
	pstm := propertysettype.NewManager(stopCh, pstCh)
	tm := thing.NewManager(stopCh,
		thing.WithThingTypeChan(ttCh),
		thing.WithThingChan(tCh),
		thing.WithPropertySetTypeManager(pstm),
		thing.WithUpdateCacheFunc(func(pst *model.PropertySetType, et model.EventType) { rollupManager.UpdateCache(pst, et) }),
	)
	rollupManager = data.NewManager(tm, store)
	pstm.InitWatch()
	rollupManager.StartWatch()

	return &config.Config{RollupManager: rollupManager}, nil
}
