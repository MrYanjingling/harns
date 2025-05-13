package instance

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/data/storage"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"lightiot/pkg/util/bitset"
	"lightiot/pkg/util/randutil"
	"time"
)

var (
	workerTableDefinition = data.NewTableDefinition(consumerDBName, make(map[string]*data.Column, 2), map[string]interface{}{data.Database: consumerDBName})
)

type registry struct {
	heartbeatInterval int64
	maxInstances      uint16
	logStorage        logstorage.Interface
	stopCh            <-chan struct{}

	instanceName string
}

func newRegistry(stopCh <-chan struct{}, heartbeatInterval int64, maxInstances uint16, logStorage logstorage.Interface) *registry {
	return &registry{
		heartbeatInterval: heartbeatInterval,
		maxInstances:      maxInstances,
		logStorage:        logStorage,
		stopCh:            stopCh,

		instanceName: randutil.StringN(3),
	}
}

func init() {
	workerTableDefinition.
		AddColumn(consumerPrimaryKey, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
		AddColumn(consumerDataColumn, data.ColumnTypeAttribute, data.ColumnDatatypeString)
}

func (r *registry) allocateIDs(lockedIDs sets.Int32) []uint16 {
	var (
		self *consumerInfo
		now  = time.Now()
	)

	ws := r.read()
	index := ws.findByName(r.instanceName)
	if index == -1 {
		self = &consumerInfo{
			IDs:        make([]uint16, 0),
			Name:       r.instanceName,
			RecentTime: now.UnixMilli(),
		}
		ws = append(ws, self)
	} else {
		self = ws[index]
		self.RecentTime = now.UnixMilli()
	}

	ws = r.collate(ws)

	r.allocate(self, ws, lockedIDs)

	r.write(now, ws)

	return self.IDs
}

func (r *registry) collate(ws consumers) consumers {
	now := time.Now().UnixMilli()
	i := 0
	for _, w := range ws {
		if now-w.RecentTime < 20*r.heartbeatInterval {
			ws[i] = w
			i++
		}
	}

	for j := i; j < len(ws); j++ {
		ws[j] = nil
	}
	ws = ws[:i]
	return ws
}

func (r *registry) allocate(self *consumerInfo, ws consumers, lockedIDs sets.Int32) {
	instanceCnt := len(ws)
	index := ws.findByName(self.Name)
	expectedIDCnt := (int(r.maxInstances) + instanceCnt - index - 1) / instanceCnt
	allocateIDCnt := expectedIDCnt - len(self.IDs)

	if allocateIDCnt < 0 {
		for i, cnt := len(self.IDs)-1, -allocateIDCnt; i >= 0 && cnt > 0; i-- {
			if !lockedIDs.Has(int32(i)) {
				self.IDs = append(self.IDs[:i], self.IDs[i+1:]...)
			}
			cnt--
		}
	} else if allocateIDCnt > 0 {
		bs := bitset.New(uint(r.maxInstances))
		for _, w := range ws {
			for _, id := range w.IDs {
				bs.Set(uint(id))
			}
		}
		for i, cnt := uint(0), allocateIDCnt; i < uint(r.maxInstances) && cnt > 0; cnt-- {
			ok := false
			if i, ok = bs.NextClear(i); ok {
				self.IDs = append(self.IDs, uint16(i))
			}
			i++
		}
	}
}

func (r *registry) read() consumers {
	td := r.getWorkerTableDefinition()
	keys := map[string]interface{}{
		consumerPrimaryKey: primaryKeyPlaceholder,
	}
	q := query.NewSingle([]*data.TableDefinition{td}, sets.NewString(storage.TimeKey, consumerDataColumn), keys, true)
	ret, _ := r.logStorage.Get(q)

	var cs consumers
	if ret == nil {
		return cs
	}

	if data, ok := ret.([]map[string]interface{}); ok {
		if len(data) > 0 {
			if err := json.Unmarshal([]byte(data[0][consumerDataColumn].(string)), &cs); err != nil {
				klog.V(3).InfoS("Failed to parse consumer info", "err", err)
			}
		}
	}
	return cs
}

func (r *registry) write(now time.Time, ws consumers) {
	wb, err := json.Marshal(ws)
	if err != nil {
		klog.V(3).InfoS("Failed to marshal consumers")
		return
	}

	td := r.getWorkerTableDefinition()
	keys := map[string]interface{}{
		consumerPrimaryKey: primaryKeyPlaceholder,
	}
	row := data.NewRow(map[string]interface{}{storage.TimeKey: now.Truncate(7 * 24 * time.Hour), consumerDataColumn: wb})
	statement := insert.NewUpsert(td, keys, []*data.Row{row}, false)
	_, _ = r.logStorage.Insert(statement)
}

func (r *registry) getWorkerTableDefinition() *data.TableDefinition {
	return workerTableDefinition
}
