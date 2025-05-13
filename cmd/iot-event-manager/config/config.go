package config

import (
	"lightiot/pkg/event"
	"lightiot/pkg/logstorage"
)

type Config struct {
	LogStorage logstorage.Interface
	EventConfig event.Config
}
