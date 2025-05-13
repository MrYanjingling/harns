package options

import (
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-installation-manager/config"
	baseoptions "lightiot/pkg/generic/options"
	"lightiot/pkg/generic/runtime"
	"lightiot/pkg/installation"
	"time"
)

const (
	_defaultRegistryUrl = "http://127.0.0.1:32110"
	_defaultPort        = "32110"
	_defaultWait        = time.Second * 15
)

type Options struct {
	RegistryUrl string        `json:"registry-url"`
	Port        string        `json:"port"`
	Wait        time.Duration `json:"graceful-timeout"`
	baseoptions.BaseOptions
}

func NewDefaultOptions() *Options {
	return &Options{
		RegistryUrl: _defaultRegistryUrl,
		Port:        _defaultPort,
		Wait:        _defaultWait,
		BaseOptions: baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVarP(&o.RegistryUrl, "registry-url", "", o.RegistryUrl, "The url of installation manager self")
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port iot-installation-manager exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
}

func (o *Options) Config(stopCh chan struct{}) (*config.Config, error) {
	iCh := make(chan runtime.Object)
	m := installation.NewManager(stopCh, iCh)
	if err := m.Init(o.RegistryUrl); err != nil {
		close(stopCh)
		return nil, err
	}
	return &config.Config{InsManager: m}, nil
}
