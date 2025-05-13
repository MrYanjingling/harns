package data

import (
	"encoding/json"
	lru "github.com/hashicorp/golang-lru"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	rupdata "lightiot/pkg/consumer/data"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/model/runtime"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	v1 "lightiot/pkg/model/v1"
	"sort"
	"time"
)

type Manager struct {
	thingManager *thing.Manager
	logStore     logstorage.Interface

	cache *lru.Cache
}

var (
	allRollupNumericFields []byte
	allRollupBoolFields    []byte
)

func init() {
	for _, v := range RollupNumericFieldFromString {
		allRollupNumericFields = append(allRollupNumericFields, byte(v))
	}

	for _, v := range RollupBoolFieldFromString {
		allRollupBoolFields = append(allRollupBoolFields, byte(v))
	}
}

func NewManager(tm *thing.Manager, logStore logstorage.Interface) *Manager {
	cache, err := lru.New(cacheSize)
	if err != nil {
		klog.InfoS("Failed to create cache", "err", err)
	}
	return &Manager{
		thingManager: tm,
		logStore:     logStore,
		cache:        cache,
	}
}

func (m *Manager) StartWatch() {
	m.thingManager.InitWatch()
}

func (m *Manager) GetThingById(id string) (*runtime.Thing, error) {
	return m.thingManager.GetThingById(id)
}

func (m *Manager) GetRollup(thingId, psName string, start, end time.Time, selects map[string][]byte, interval Interval, limit int, pst *model.PropertySetType) []map[string]interface{} {
	var tb job.TimeBlockUnit
	d := interval.toDuration()
	switch {
	case d >= 24*time.Hour, d == 0:
		tb = job.TimeBlockUnitDay
	case d >= time.Hour:
		tb = job.TimeBlockUnitHour
	case d >= time.Minute:
		tb = job.TimeBlockUnitMinute
	default:
		klog.V(2).InfoS("Invalid interval", "interval", interval)
		return nil
	}

	properties := sets.String{}
	for k := range selects {
		properties.Insert(k)
	}

	var rups map[string]interface{}
	durationMs := int64(d / time.Millisecond)
	if durationMs > job.TimeBlockUnitDuration[tb] || durationMs == 0 {
		rups = m.getRollupsOnTheFly(thingId, psName, start, end, string(job.TimeBlockUnitToString[tb]), durationMs, interval.amount, properties, pst)
	} else {
		rups = m.getRollups(thingId, psName, start, end, string(job.TimeBlockUnitToString[tb]), properties, pst)
	}

	ret := m.extractData(rups, start, end, interval, selects, pst.PropertyByName)

	return ret
}

func (m *Manager) getRollupsOnTheFly(thingId, psName string, start, end time.Time, tb string, durationMs int64, interval int, properties sets.String, pst *model.PropertySetType) map[string]interface{} {
	td := m.getCommandTableDefinition(pst)
	return rupdata.GetRollupsOnTheFly(thingId, psName, start, end, tb, durationMs, interval, properties, td, pst, m.logStore)
}

func (m *Manager) getRollups(thingId, psName string, start, end time.Time, tb string, properties sets.String, pst *model.PropertySetType) map[string]interface{} {
	td := m.getCommandTableDefinition(pst)
	return rupdata.GetRollups(thingId, psName, start, end, tb, properties, td, m.logStore)
}

func (m *Manager) getCommandTableDefinition(pst *model.PropertySetType) *data.TableDefinition {
	if v, ok := m.cache.Get(pst.ID); ok {
		return v.(*data.TableDefinition)
	}
	td := rupdata.GetCommandTableDefinition(pst)
	m.cache.Add(pst.ID, td)
	return td
}

func (m *Manager) UpdateCache(pst *model.PropertySetType, et model.EventType) {
	if et == model.Remove {
		m.cache.Remove(pst.ID)
	} else if et == model.Update {
		m.cache.Add(pst.ID, rupdata.GetCommandTableDefinition(pst))
	}
}

func (m *Manager) extractData(rups map[string]interface{}, start, end time.Time, interval Interval, selects map[string][]byte, propertiesByName map[string]*model.Property) []map[string]interface{} {
	rollups := make(map[string][]*RollupRecord)
	for k, v := range rups {
		if propertiesByName[k].DataType != v1.DataTypeBoolean {
			rr := extractNumeric(v.([]data.ResultColumn), selects[k])
			rollups[k] = rr
		} else {
			rr := extractBool(v.([]data.ResultColumn), selects[k])
			rollups[k] = rr
		}
	}

	groupByTime := make(map[time.Time]map[string]interface{})

	for k, v := range rollups {
		for _, r := range v {
			if _, ok := groupByTime[r.end]; !ok {
				groupByTime[r.end] = make(map[string]interface{})
			}
			groupByTime[r.end][k] = r.item
		}
	}

	keys := make([]time.Time, 0, len(groupByTime))
	for k := range groupByTime {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	ret := make([]map[string]interface{}, len(groupByTime))
	isFixDuration := true
	durationMs := int64(interval.toDuration() / time.Millisecond)
	if durationMs == 0 {
		isFixDuration = false
	}
	for i, k := range keys {
		v := groupByTime[k]
		if !isFixDuration {
			year, month, _ := k.Date()
			durationMs = 0
			j := 0
			for j < interval.amount {
				currentMonth := int(month) - i - 1
				if currentMonth <= 0 {
					currentMonth += len(job.DaysPerMonth)
				}
				days := job.DaysPerMonth[currentMonth-1]
				if currentMonth == 2 && (year%4 == 0 && (year%100 != 0 || year%400 == 0)) {
					days = 29
				}
				durationMs += int64(days) * job.TimeBlockUnitDuration[job.TimeBlockUnitDay]
				j++
			}
		}
		v[apis.Start] = k.UTC().Add(-time.Duration(durationMs) * time.Millisecond)
		v[apis.End] = k.UTC()
		ret[i] = v
	}

	return ret
}

func extractNumeric(data []data.ResultColumn, fields []byte) []*RollupRecord {
	if len(data) == 0 {
		return nil
	}

	cnt := len(data)
	items := make([]*RollupRecord, cnt)
	for i, c := range data {
		rni, ok := c.Value.(*rupdata.RollupNumericItem)
		if !ok {
			item := rupdata.RollupNumericItem{}
			if err := json.Unmarshal([]byte(c.Value.(string)), &item); err != nil {
				klog.V(3).InfoS("Failed to parse rollup item", "err", err)
				continue
			}
			rni = &item
		}
		rr := make(map[string]interface{})
		for _, f := range fields {
			rr = RollupNumericField(f).Assemble(rr, rni)
		}
		items[i] = &RollupRecord{
			end:  c.Time,
			item: rr,
		}
	}
	return items
}

func extractBool(data []data.ResultColumn, fields []byte) []*RollupRecord {
	if len(data) == 0 {
		return nil
	}

	cnt := len(data)
	items := make([]*RollupRecord, cnt)
	for i, c := range data {
		rbi, ok := c.Value.(*rupdata.RollupBoolItem)
		if !ok {
			item := rupdata.RollupBoolItem{}
			if err := json.Unmarshal([]byte(c.Value.(string)), &item); err != nil {
				klog.V(3).InfoS("Failed to parse rollup item", "err", err)
				continue
			}
			rbi = &item
		}

		rr := make(map[string]interface{})
		for _, f := range fields {
			rr = RollupBoolField(f).Assemble(rr, rbi)
		}
		items[i] = &RollupRecord{
			end:  c.Time,
			item: rr,
		}
	}
	return items
}

type numericMapFunc func(*rupdata.RollupNumericItem) interface{}

var numericMapFuncs = map[RollupNumericField]numericMapFunc{
	RollupNumericFieldFirstTime:    func(rni *rupdata.RollupNumericItem) interface{} { return rni.FirstTime },
	RollupNumericFieldFirstValue:   func(rni *rupdata.RollupNumericItem) interface{} { return rni.FirstValue },
	RollupNumericFieldLastTime:     func(rni *rupdata.RollupNumericItem) interface{} { return rni.LastTime },
	RollupNumericFieldLastValue:    func(rni *rupdata.RollupNumericItem) interface{} { return rni.LastValue },
	RollupNumericFieldMinTime:      func(rni *rupdata.RollupNumericItem) interface{} { return rni.MinTime },
	RollupNumericFieldMinValue:     func(rni *rupdata.RollupNumericItem) interface{} { return rni.MinValue },
	RollupNumericFieldMaxTime:      func(rni *rupdata.RollupNumericItem) interface{} { return rni.MaxTime },
	RollupNumericFieldMaxValue:     func(rni *rupdata.RollupNumericItem) interface{} { return rni.MaxValue },
	RollupNumericFieldAvg:          func(rni *rupdata.RollupNumericItem) interface{} { return rni.Avg },
	RollupNumericFieldSum:          func(rni *rupdata.RollupNumericItem) interface{} { return rni.Sum },
	RollupNumericFieldSD:           func(rni *rupdata.RollupNumericItem) interface{} { return rni.SD },
	RollupNumericFieldGoodCnt:      func(rni *rupdata.RollupNumericItem) interface{} { return rni.GoodCnt },
	RollupNumericFieldBadCnt:       func(rni *rupdata.RollupNumericItem) interface{} { return rni.BadCnt },
	RollupNumericFieldUncertainCnt: func(rni *rupdata.RollupNumericItem) interface{} { return rni.UncertainCnt },
}

func (rnf RollupNumericField) Assemble(rr map[string]interface{}, rni *rupdata.RollupNumericItem) map[string]interface{} {
	s := RollupNumericFieldToString[rnf]
	rr[s] = numericMapFuncs[rnf](rni)
	return rr
}

type boolMapFunc func(*rupdata.RollupBoolItem) interface{}

var boolMapFuncs = map[RollupBoolField]boolMapFunc{
	RollupBoolFieldFirstTrueTime:  func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.FirstTrueTime },
	RollupBoolFieldFirstFalseTime: func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.FirstFalseTime },
	RollupBoolFieldLastTrueTime:   func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.LastTrueTime },
	RollupBoolFieldLastFalseTime:  func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.LastFalseTime },
	RollupBoolFieldLastValue:      func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.LastValue },
	RollupBoolFieldTrueCnt:        func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.TrueCnt },
	RollupBoolFieldTrueDuration:   func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.TrueDuration },
	RollupBoolFieldFalseCnt:       func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.FalseCnt },
	RollupBoolFieldFalseDuration:  func(rbi *rupdata.RollupBoolItem) interface{} { return rbi.FalseDuration },
}

func (rbf RollupBoolField) Assemble(rr map[string]interface{}, rbi *rupdata.RollupBoolItem) map[string]interface{} {
	s := RollupBoolFieldToString[rbf]
	rr[s] = boolMapFuncs[rbf](rbi)
	return rr
}
