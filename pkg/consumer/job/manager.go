package job

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"strconv"
	"sync"
	"time"
)

type Manager struct {
	maxInstances uint16
	delaySec     int64
	refreshSec   uint16
	logStorage   logstorage.Interface
	stopCh       <-chan struct{}

	tds map[Group]*data.TableDefinition
	// the reason that gives up ristretto.Cache is the issue https://github.com/kubernetes/kubernetes/issues/61006
	// which has not been fixed by klog https://github.com/kubernetes/klog#why-was-klog-created
	// TODO: hierarchical cache, 1 min/1 hour/1 day should have different clean period
	cache *sync.Map // procTime -> *sync.Map
	// cache *ristretto.Cache
}

func New(stopCh <-chan struct{}, maxInstances uint16, delaySec int64, enableCache bool, refreshSec uint16, logStorage logstorage.Interface) *Manager {
	tds := make(map[Group]*data.TableDefinition, len(GroupToString))
	for jg, name := range GroupToString {
		tds[jg] = data.NewTableDefinition(name, make(map[string]*data.Column), map[string]interface{}{data.Database: jobDBName})
		tds[jg].AddColumn(jobWorkerID, data.ColumnTypePrimaryKey, data.ColumnDatatypeInt).
			AddColumn(jobData, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
			AddColumn(jobProcTime, data.ColumnTypeAttribute, data.ColumnDatatypeTimestamp).
			AddColumn(jobPlaceHolder, data.ColumnTypeAttribute, data.ColumnDatatypeString)
	}
	m := &Manager{
		maxInstances: maxInstances,
		delaySec:     delaySec,
		refreshSec:   refreshSec,
		logStorage:   logStorage,
		stopCh:       stopCh,
		tds:          tds,
		cache:        &sync.Map{},
	}

	if enableCache {
		go m.maintainCache()
	}

	return m
}

func (m *Manager) RetrieveJobs(ids []uint16, ch chan<- *Job, group Group) int {
	var cnt int
	for _, id := range ids {
		keys := map[string]interface{}{
			jobWorkerID: id,
		}
		now := time.Now()

		q := query.NewRange([]*data.TableDefinition{m.tds[group]}, now.Add(-7*24*time.Hour), now, sets.NewString(jobWorkerID, jobData, jobPlaceHolder), keys, 2000, false, true, func() string { return jobGroup })

		data, _ := m.logStorage.List(q)
		items, _ := data.([]map[string]interface{})
		cnt += len(items)

		for _, d := range items {
			var j Job
			if err := j.Of(d[jobData].(string)); err != nil {
				klog.V(2).InfoS("Failed to parse job data", "err", err)
			}
			j.ProcTimeSec = d[jobProcTime].(time.Time).Unix()

			j.Group = GroupFromString[d[jobGroup].(string)]

			// TODO: how clickhouse handle it
			// influxdb: Measurements, tag keys, tag values, and field keys are always strings.
			consumerId, _ := strconv.Atoi(d[jobWorkerID].(string))
			j.ConsumerID = uint16(consumerId)

			ch <- &j
		}
	}
	return cnt
}

func (m *Manager) writeJob(j *Job, cache bool) error {
	if cache {
		var cacheKeys *sync.Map

		j.ProcTimeSec += j.DelaySec(m.delaySec)
		k := j.TimeBlock.String() + j.Key
		v, ok := m.cache.Load(j.ProcTimeSec)
		if ok {
			cacheKeys = v.(*sync.Map)
			if _, ok := cacheKeys.Load(k); ok {
				klog.V(5).InfoS("Cached job", "procTime", j.ProcTimeSec, "key", k)
				return nil
			}
		} else {
			cacheKeys = &sync.Map{}
		}
		cacheKeys.Store(k, struct{}{})
		m.cache.Store(j.ProcTimeSec, cacheKeys)
	}

	keys := map[string]interface{}{
		jobWorkerID: j.ConsumerID,
		jobData:     j.String(),
	}
	row := data.NewRow(map[string]interface{}{jobProcTime: time.Unix(j.ProcTimeSec, 0), jobPlaceHolder: ""})
	statement := insert.NewUpsert(m.tds[j.Group], keys, []*data.Row{row}, false)
	_, _ = m.logStorage.Insert(statement)

	klog.V(5).InfoS("Write job", "job", j, "procTime", j.ProcTimeSec)
	return nil
}

func (m *Manager) DeleteJob(j *Job) {
	keys := map[string]interface{}{
		jobWorkerID: j.ConsumerID,
		jobData:     j.String(),
	}
	t := time.Unix(j.ProcTimeSec, 0)
	statement := insert.NewDelete(m.tds[j.Group], t, t, keys)
	if _, err := m.logStorage.Delete(statement); err != nil {
		klog.V(1).InfoS("Failed to delete job", "job", j, "err", err)
	}
}

func (m *Manager) Rollup1MinuteJob(key string, now, start time.Time, loc *time.Location) error {
	j := rollup1MinuteJob(key, now, start, loc, m.maxInstances)
	return m.writeJob(&j, true)
}

func (m *Manager) Rollup1HourJob(key string, now, start time.Time, loc *time.Location) error {
	j := rollup1HourJob(key, now, start, loc, m.maxInstances)
	return m.writeJob(&j, true)
}

func (m *Manager) Rollup1DayJob(key string, now, start time.Time, loc *time.Location) error {
	j := rollup1DayJob(key, now, start, loc, m.maxInstances)
	return m.writeJob(&j, true)
}

func (m *Manager) DeleteThingJob(key string, now time.Time) error {
	j := deleteThingJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) DeleteEventType(key string, now time.Time) error {
	j := deleteEventTypeJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) DeleteCommandType(key string, now time.Time) error {
	j := deleteCommandTypeJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) DeleteThingEvent(key string, now time.Time) error {
	j := deleteThingEventJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) DeleteThingCommand(key string, now time.Time) error {
	j := deleteThingCommandJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) DeleteThingAction(key string, now time.Time) error {
	j := deleteThingActionJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) DeleteThingTimeSeries(key string, now time.Time) error {
	j := deleteThingTimeSeriesJob(key, now, m.maxInstances)
	return m.writeJob(&j, false)
}

func (m *Manager) maintainCache() {
	// TODO: hierarchical cache, 1 min/1 hour/1 day should have different clean period
	ticker := time.NewTicker(time.Duration(m.refreshSec) * time.Second)
	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()
			m.cache.Range(func(key, value interface{}) bool {
				if now > key.(int64) {
					m.cache.Delete(key)
				}
				return true
			})
		case <-m.stopCh:
			ticker.Stop()
			klog.V(2).InfoS("Stopped maintain cache")
			return
		}
	}
}
