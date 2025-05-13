package config

import (
	"lightiot/pkg/client"
	"lightiot/pkg/control"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/logstorage"
)

type Config struct {
	IotBrokerClient client.Client
	LogStorage      logstorage.Interface
	Config          *runtime.Config
	TTL             int
	ControlConfig   control.Config
}
