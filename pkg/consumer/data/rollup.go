package data

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	model "lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"lightiot/pkg/util/strutil"
	"math"
	"reflect"
	"time"
)

func (w *Worker) Rollup(j *job.Job) {
	thingId, psName := strutil.Split(j.Key)

	thing, err := w.tm.GetThingById(thingId)
	if err != nil {
		klog.V(3).InfoS("Thing not found", "id", thingId)
		return
	}
	ps, ok := thing.PropertySetByName[psName]
	if !ok {
		klog.V(3).InfoS("PropertySet not found", "propertySet", psName, "thingId", thingId)
		return
	}

	propertiesByName := ps.PropertySetType.PropertyByName
	legalProperties := sets.String{}
	for k, v := range propertiesByName {
		if v.DataType != v1.DataTypeString {
			legalProperties.Insert(k)
		}
	}

	if legalProperties.Len() == 0 {
		klog.V(3).InfoS("There was no legal property")
		return
	}

	start := j.TimeBlock.ToTime()
	end := start.Add(j.TimeBlock.Duration())
	if j.TimeBlock.IsMinimalInterval() {
		raw := w.getRawTimeSeries(thingId, psName, ps.PropertySetType.ID, start, end, legalProperties.UnsortedList())
		rup := w.aggregateData(raw, start, end, propertiesByName)
		if len(rup) > 0 {
			w.persistRollup(thingId, psName, end, j.TimeBlock.GetUnit(), rup, ps.PropertySetType)
		}
	} else {
		tb := job.TimeBlockUnitMinute
		if j.TimeBlock.Duration() > time.Hour {
			tb = job.TimeBlockUnitHour
		}

		rups := w.getRollups(thingId, psName, start, end, string(job.TimeBlockUnitToString[tb]), legalProperties, ps.PropertySetType)
		rup := w.rollupData(rups, start, end, job.TimeBlockUnitDuration[tb], propertiesByName)
		w.persistRollup(thingId, psName, end, j.TimeBlock.GetUnit(), rup, ps.PropertySetType)
	}
}

func (w *Worker) aggregateData(data map[string][]field, start, end time.Time, propertiesByName map[string]*model.Property) map[string]interface{} {
	rollups := make(map[string]interface{})
	for k, v := range data {
		if propertiesByName[k].DataType != v1.DataTypeBoolean {
			if r := aggregateNumeric(v); r != nil {
				rollups[k] = r
				klog.V(5).InfoS("Aggregated numeric", "property", k, "result", r)
			}
		} else {
			if r := aggregateBool(v, start, end); r != nil {
				rollups[k] = r
				klog.V(5).InfoS("Aggregated bool", "property", k, "result", r)
			}
		}
	}
	return rollups
}

func (w *Worker) rollupData(rups map[string]interface{}, start, end time.Time, durationMs int64, propertiesByName map[string]*model.Property) map[string]interface{} {
	rollups := make(map[string]interface{})
	for k, v := range rups {
		if propertiesByName[k].DataType != v1.DataTypeBoolean {
			r := rollupNumeric(v.([]data.ResultColumn))
			rollups[k] = r
			klog.V(5).InfoS("Rollup numeric", "property", k, "result", r)
		} else {
			r := rollupBool(v.([]data.ResultColumn), start, end, durationMs)
			rollups[k] = r
			klog.V(5).InfoS("Rollup bool", "property", k, "result", r)
		}
	}
	return rollups
}

func (w *Worker) persistRollup(thingId, psName string, end time.Time, tb string, rup map[string]interface{}, pst *model.PropertySetType) {
	td := w.getCommandTableDefinition(pst)

	keys := map[string]interface{}{
		columnThingId:         thingId,
		columnPropertySetName: psName,
		columnTimeBlock:       tb,
	}
	row := &data.Row{}
	row.SetValue(data.TimeKey, end)
	for k, v := range rup {
		s, err := json.Marshal(v)
		if err != nil {
			klog.V(4).InfoS("Failed to marshal rollup item", "err", err)
			continue
		}
		row.SetValue(k, s)
	}

	statement := insert.NewUpsert(td, keys, []*data.Row{row}, false)

	_, _ = w.logStore.Insert(statement)
}

func (w *Worker) getRollups(thingId, psName string, start, end time.Time, tb string, properties sets.String, pst *model.PropertySetType) map[string]interface{} {
	td := w.getCommandTableDefinition(pst)
	return GetRollups(thingId, psName, start, end, tb, properties, td, w.logStore)
}

func (w *Worker) getCommandTableDefinition(pst *model.PropertySetType) *data.TableDefinition {
	if v, ok := w.cache.Get(pst.ID); ok {
		return v.(*data.TableDefinition)
	}
	td := GetCommandTableDefinition(pst)
	w.cache.Add(pst.ID, td)
	return td
}

func (w *Worker) updateCache(pst *model.PropertySetType, et model.EventType) {
	if et == model.Remove {
		w.cache.Remove(pst.ID)
	} else if et == model.Update {
		w.cache.Add(pst.ID, GetCommandTableDefinition(pst))
	}
}

func GetCommandTableDefinition(pst *model.PropertySetType) *data.TableDefinition {
	td := data.NewTableDefinition(pst.ID, make(map[string]*data.Column), map[string]interface{}{data.Database: timeSeriesRupDataBucket})

	td.AddColumn(columnThingId, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
		AddColumn(columnPropertySetName, data.ColumnTypePrimaryKey, data.ColumnDatatypeString).
		AddColumn(columnTimeBlock, data.ColumnTypePrimaryKey, data.ColumnDatatypeString)

	for _, p := range pst.Properties {
		if p.DataType == v1.DataTypeInt {
			td.AddColumn(p.Name, data.ColumnTypeAttribute, data.ColumnDatatypeInt)
		} else if p.DataType == v1.DataTypeLong {
			td.AddColumn(p.Name, data.ColumnTypeAttribute, data.ColumnDatatypeLong)
		} else if p.DataType == v1.DataTypeDouble {
			td.AddColumn(p.Name, data.ColumnTypeAttribute, data.ColumnDatatypeDouble)
		} else if p.DataType == v1.DataTypeBoolean {
			td.AddColumn(p.Name, data.ColumnTypeAttribute, data.ColumnDatatypeBoolean)
		}
	}
	return td
}

func GetRollupsOnTheFly(thingId, psName string, start, end time.Time, tb string, durationMs int64, interval int, properties sets.String, td *data.TableDefinition, pst *model.PropertySetType, logStore logstorage.Interface) map[string]interface{} {
	records := make(map[string]interface{})
	record := data.ResultColumn{}
	isFixDuration := true
	if durationMs == 0 {
		isFixDuration = false
	}
	for start.Before(end) {
		if !isFixDuration {
			year, month, _ := end.Date()
			durationMs = 0
			i := 0
			for i < interval {
				currentMonth := int(month) - i - 1
				if currentMonth <= 0 {
					currentMonth += len(job.DaysPerMonth)
				}
				days := job.DaysPerMonth[currentMonth-1]
				if currentMonth == 2 && (year%4 == 0 && (year%100 != 0 || year%400 == 0)) {
					days = 29
				}
				durationMs += int64(days) * job.TimeBlockUnitDuration[job.TimeBlockUnitDay]
				i++
			}
		}
		tmpStart := end.Add(-time.Duration(durationMs) * time.Millisecond)
		rups := GetRollups(thingId, psName, tmpStart, end, tb, properties, td, logStore)
		for k, v := range rups {
			if pst.PropertyByName[k].DataType != v1.DataTypeBoolean {
				record = data.ResultColumn{Value: rollupNumeric(v.([]data.ResultColumn)), Time: end}
			} else {
				record = data.ResultColumn{Value: rollupBool(v.([]data.ResultColumn), tmpStart, end, durationMs), Time: end}
			}
			r, ok := records[k].([]data.ResultColumn)
			if !ok {
				r = []data.ResultColumn{}
			}
			r = append(r, record)
			records[k] = r
		}
		end = tmpStart
	}

	return records
}

func GetRollups(thingId, psName string, start, end time.Time, tb string, properties sets.String, td *data.TableDefinition, logStore logstorage.Interface) map[string]interface{} {
	keys := map[string]interface{}{
		columnThingId:         thingId,
		columnPropertySetName: psName,
		columnTimeBlock:       tb,
	}

	q := query.NewRange([]*data.TableDefinition{td}, start.Add(time.Millisecond), end.Add(time.Millisecond), properties, keys, 2000, false, false, nil)
	ret, _ := logStore.List(q)

	rups := ret.([]map[string]interface{})

	if len(rups) == 0 {
		klog.V(4).InfoS("Failed to get rollup", "thingId", thingId, "psName", psName, "start", start, "end", end, "timeBlock", tb)
		return nil
	}
	return rups[0]
}

func aggregateNumeric(data []field) *RollupNumericItem {
	if len(data) == 0 {
		return nil
	}

	cnt := len(data)
	r := &RollupNumericItem{
		FirstTime:    data[0].time,
		FirstValue:   data[0].value,
		LastTime:     data[cnt-1].time,
		LastValue:    data[cnt-1].value,
		Avg:          0,
		Sum:          0,
		SD:           0,
		GoodCnt:      int64(cnt),
		BadCnt:       0,
		UncertainCnt: 0,
	}

	var (
		minValue = toFloat64(data[0].value)
		maxValue = minValue
		minIndex = 0
		maxIndex = minIndex
	)
	for i, f := range data {
		v := toFloat64(f.value)
		if v < minValue {
			minValue = v
			minIndex = i
		}
		if v > maxValue {
			maxValue = v
			maxIndex = i
		}
		r.Sum += v
	}
	r.MinTime = data[minIndex].time
	r.MinValue = data[minIndex].value
	r.MaxTime = data[maxIndex].time
	r.MaxValue = data[maxIndex].value
	r.Avg = r.Sum / float64(cnt)

	mean := r.Avg
	var sq float64 // sum of square
	for _, f := range data {
		sq += math.Pow(toFloat64(f.value)-mean, 2)
	}
	r.SD = math.Sqrt(sq / float64(cnt))

	return r
}

func aggregateBool(data []field, start, end time.Time) *RollupBoolItem {
	if len(data) == 0 {
		return nil
	}

	r := &RollupBoolItem{
		FirstTrueTime:  time.Time{},
		FirstFalseTime: time.Time{},
		LastTrueTime:   time.Time{},
		LastFalseTime:  time.Time{},
		LastValue:      data[len(data)-1].value.(bool),
		TrueCnt:        0,
		TrueDuration:   0,
		FalseCnt:       0,
		FalseDuration:  0,
	}
	var (
		cnts      = make(map[bool]int64, 2)
		durations = make(map[bool]time.Duration, 2)
		firsts    = make(map[bool]time.Time, 2)
		trues     = make([]time.Time, 0, len(data))
		falses    = make([]time.Time, 0, len(data))
	)

	sentry := data[0].value.(bool)
	firsts[sentry] = data[0].time
	durations[sentry] = data[0].time.Sub(start)
	for _, f := range data {
		v := f.value.(bool)

		if v != sentry {
			firsts[v] = f.time
			durations[!v] += f.time.Sub(firsts[!v])
			sentry = v
		}

		if v {
			trues = append(trues, f.time)
		} else {
			falses = append(falses, f.time)
		}
		cnts[v]++
	}

	if cnts[true] > 0 {
		r.FirstTrueTime = trues[0]
		r.LastTrueTime = trues[cnts[true]-1]
	}

	if cnts[false] > 0 {
		r.FirstFalseTime = falses[0]
		r.LastFalseTime = falses[cnts[false]-1]
	}

	durations[r.LastValue] += end.Sub(firsts[r.LastValue])

	r.TrueCnt = cnts[true]
	r.FalseCnt = cnts[false]

	if r.TrueCnt == 0 {
		r.FalseDuration = end.Sub(start).Milliseconds()
	} else if r.FalseCnt == 0 {
		r.TrueDuration = end.Sub(start).Milliseconds()
	} else {
		r.TrueDuration = durations[true].Milliseconds()
		r.FalseDuration = durations[false].Milliseconds()
	}

	return r
}

func rollupNumeric(data []data.ResultColumn) *RollupNumericItem {
	if len(data) == 0 {
		return nil
	}

	cnt := len(data)
	items := make([]*RollupNumericItem, cnt)
	for i, c := range data {
		var rni RollupNumericItem
		if err := json.Unmarshal([]byte(c.Value.(string)), &rni); err != nil {
			klog.V(3).InfoS("Failed to parse rollup item", "err", err)
			continue
		}
		items[i] = &rni
	}

	r := &RollupNumericItem{
		FirstTime:    items[0].FirstTime,
		FirstValue:   items[0].FirstValue,
		LastTime:     items[cnt-1].LastTime,
		LastValue:    items[cnt-1].LastValue,
		Avg:          0,
		Sum:          0,
		SD:           0,
		GoodCnt:      0,
		BadCnt:       0,
		UncertainCnt: 0,
	}

	var (
		minValue = toFloat64(items[0].MinValue)
		maxValue = toFloat64(items[0].MaxValue)
		minIndex = 0
		maxIndex = minIndex
	)
	for i, f := range items {
		minV := toFloat64(f.MinValue)
		maxV := toFloat64(f.MaxValue)
		if minV < minValue {
			minValue = minV
			minIndex = i
		}
		if maxV > maxValue {
			maxValue = maxV
			maxIndex = i
		}
		r.Sum += toFloat64(f.Sum)
		r.GoodCnt += f.GoodCnt
		r.BadCnt += f.BadCnt
		r.UncertainCnt += f.UncertainCnt
	}
	r.MinTime = items[minIndex].MinTime
	r.MinValue = items[minIndex].MinValue
	r.MaxTime = items[maxIndex].MaxTime
	r.MaxValue = items[maxIndex].MaxValue
	r.Avg = r.Sum / float64(r.GoodCnt+r.BadCnt+r.UncertainCnt)

	mean := r.Avg
	var sq float64 // sum of square
	for _, f := range items {
		sq += float64(f.GoodCnt+f.BadCnt+f.UncertainCnt) * (math.Pow(f.SD, 2) + math.Pow(f.Avg-mean, 2))
	}
	r.SD = math.Sqrt(sq / float64(r.GoodCnt+r.BadCnt+r.UncertainCnt))

	return r
}

func rollupBool(data []data.ResultColumn, start, end time.Time, durationMs int64) *RollupBoolItem {
	if len(data) == 0 {
		return nil
	}

	cnt := len(data)
	items := make([]*RollupBoolItem, cnt)
	for i, c := range data {
		var rnb RollupBoolItem
		if err := json.Unmarshal([]byte(c.Value.(string)), &rnb); err != nil {
			klog.V(3).InfoS("Failed to parse rollup item", "err", err)
			continue
		}
		rnb.endTime = c.Time
		rnb.startTime = c.Time.Add(-time.Duration(durationMs) * time.Millisecond)
		items[i] = &rnb
	}

	r := &RollupBoolItem{
		FirstTrueTime:  time.Time{},
		FirstFalseTime: time.Time{},
		LastTrueTime:   time.Time{},
		LastFalseTime:  time.Time{},
		LastValue:      items[cnt-1].LastValue,
		TrueCnt:        0,
		TrueDuration:   0,
		FalseCnt:       0,
		FalseDuration:  0,
	}

	var (
		trues     = make([]time.Time, 0, len(data))
		falses    = make([]time.Time, 0, len(data))
		durations = make(map[bool]int64, 2)
	)

	var (
		lastValue   bool
		lastEndTime time.Time
	)
	for i, f := range items {
		firstValue := false
		firstTime := f.FirstFalseTime
		firstStartTime := f.startTime
		if firstTime.IsZero() || (!f.FirstTrueTime.IsZero() && f.FirstTrueTime.Before(f.FirstFalseTime)) {
			firstValue = true
			firstTime = f.FirstTrueTime
		}
		if i == 0 {
			// nop
		} else if firstValue == lastValue {
			durations[firstValue] += firstStartTime.Sub(lastEndTime).Milliseconds()
		} else {
			durations[lastValue] += firstTime.Sub(lastEndTime).Milliseconds()
			durations[firstValue] -= firstTime.Sub(firstStartTime).Milliseconds()
		}

		if r.FirstTrueTime.IsZero() && !f.FirstTrueTime.IsZero() {
			r.FirstTrueTime = f.FirstTrueTime
		}

		if r.FirstFalseTime.IsZero() && !f.FirstFalseTime.IsZero() {
			r.FirstFalseTime = f.FirstFalseTime
		}

		if !f.LastTrueTime.IsZero() {
			trues = append(trues, f.LastTrueTime)
		}

		if !f.LastFalseTime.IsZero() {
			falses = append(falses, f.LastFalseTime)
		}

		r.TrueCnt += f.TrueCnt
		r.FalseCnt += f.FalseCnt
		durations[true] += f.TrueDuration
		durations[false] += f.FalseDuration

		lastValue = f.LastValue
		lastEndTime = f.endTime
	}

	if len(trues) > 0 {
		r.LastTrueTime = trues[len(trues)-1]
	}

	if len(falses) > 0 {
		r.LastFalseTime = falses[len(falses)-1]
	}

	if !r.FirstTrueTime.IsZero() && (r.FirstFalseTime.IsZero() || r.FirstTrueTime.Before(r.FirstFalseTime)) {
		durations[true] += items[0].startTime.Sub(start).Milliseconds()
	} else {
		durations[false] += items[0].startTime.Sub(start).Milliseconds()
	}

	durations[r.LastValue] += end.Sub(items[len(items)-1].endTime).Milliseconds()

	r.TrueDuration = durations[true]
	r.FalseDuration = durations[false]

	return r
}

func toFloat64(v interface{}) float64 {
	switch i := v.(type) {
	case float64:
		return i
	case int64:
		return float64(i)
	case int:
		return float64(i)
	default:
		klog.V(2).InfoS("Non-numeric type could not be converted to float", "type", reflect.TypeOf(v), "value", v)
		return 0
	}
}
