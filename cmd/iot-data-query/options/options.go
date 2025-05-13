package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-data-query/config"
	"lightiot/pkg/data/storage"
	baseoptions "lightiot/pkg/generic/options"
	"time"
)

const (
	_defaultInfluxDBUrl   = "http://127.0.0.1:8086"
	_defaultInfluxDBToken = ""
	_defaultPort          = "32300"
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
	fs.StringVarP(&o.InfluxDBUrl, "influxdb-url", "", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVarP(&o.InfluxDBToken, "influxdb-token", "", o.InfluxDBToken, "The InfluxDB token")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-data-query exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
}

func (o *Options) Config(stopCh <-chan struct{}) *config.Config {
	s := storage.NewStore(stopCh, o.InfluxDBUrl, o.InfluxDBToken)
	s.Init()
	return &config.Config{Store: s}
}
