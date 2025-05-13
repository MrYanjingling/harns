package insert

import "lightiot/pkg/logstorage/data"

type Upsert struct {
	*data.TableDefinition
	keys map[string]interface{}
	rows []*data.Row
	sync bool
}

func NewUpsert(td *data.TableDefinition, keys map[string]interface{}, rows []*data.Row, sync bool) *Upsert {
	return &Upsert{
		TableDefinition: td,
		keys:            keys,
		rows:            rows,
		sync:            sync,
	}
}

func (us *Upsert) IsSync() bool {
	return us.sync
}

func (us *Upsert) GetKeys() map[string]interface{} {
	return us.keys
}

func (us *Upsert) GetRows() []*data.Row {
	return us.rows
}

func (us *Upsert) GetTableDefinition() *data.TableDefinition {
	return us.TableDefinition
}
