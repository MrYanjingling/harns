package insert

import (
	"lightiot/pkg/logstorage/data"
	"time"
)

type Delete struct {
	*data.TableDefinition
	keys map[string]interface{}
	*data.Range
}

func NewDelete(td *data.TableDefinition, start, end time.Time, keys map[string]interface{}) *Delete {
	return &Delete{
		TableDefinition: td,
		keys:            keys,
		Range:           data.NewRange(start, end),
	}
}

func (d *Delete) GetKeys() map[string]interface{} {
	return d.keys
}
