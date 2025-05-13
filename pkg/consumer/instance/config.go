package instance

import (
	"lightiot/pkg/consumer/data"
	"lightiot/pkg/logstorage"
)

const (
	consumerDBName     = "consumers"
	consumerPrimaryKey = "pk"
	consumerDataColumn = "data"

	primaryKeyPlaceholder = "pl"
)

type Config struct {
	HeartbeatInterval   int64
	RetrieveJobInterval int64
	MaxInstances        uint16
	RollupWorkers       uint16
	RollupBufferSize    uint32
	DeleteWorkers       uint16
	DeleteBufferSize    uint32
	LogStorage          logstorage.Interface

	DataWorkerConfig data.Config
}
