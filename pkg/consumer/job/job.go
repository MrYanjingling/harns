package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strconv"
	"time"
)

type timeBlock struct {
	index int64
	unit  TimeBlockUnit
}

func toTimeBlock(t time.Time, unit TimeBlockUnit, loc *time.Location) timeBlock {
	lt := t.In(loc)
	year, month, day := lt.Date()
	hour, min, sec := lt.Clock()
	switch unit {
	case TimeBlockUnitDay:
		hour = 0
		fallthrough
	case TimeBlockUnitHour:
		min = 0
		fallthrough
	case TimeBlockUnitMinute:
		sec = 0
	default:
		// nop
	}

	return timeBlock{
		index: time.Date(year, month, day, hour, min, sec, 0, loc).Unix(),
		unit:  unit,
	}
}

func (tb *timeBlock) ToTime() time.Time {
	return time.Unix(tb.index, 0).UTC()
}

func (tb *timeBlock) Duration() time.Duration {
	return time.Duration(TimeBlockUnitDuration[tb.unit]) * time.Millisecond
}

func (tb *timeBlock) IsMinimalInterval() bool {
	return tb.unit == TimeBlockUnitMinute
}

func (tb *timeBlock) GetUnit() string {
	return string(TimeBlockUnitToString[tb.unit])
}

func (tb *timeBlock) String() string {
	// return strconv.FormatInt(tb.index, 10) + string(TimeBlockUnitToString[tb.unit])
	return fmt.Sprintf("%010d", tb.index) + string(TimeBlockUnitToString[tb.unit])
}

func (tb *timeBlock) Of(s string) error {
	unit := s[len(s)-1]
	index := s[:len(s)-1]

	i, err := strconv.ParseInt(index, 10, 64)
	if err != nil {
		return err
	}

	v, ok := TimeBlockUnitFromString[unit]
	if !ok {
		return fmt.Errorf("unknown datatype %s", s)
	}
	*tb = timeBlock{
		index: i,
		unit:  v,
	}
	return nil
}

func (tb timeBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(tb.String())
}

func (tb *timeBlock) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}
	var alias timeBlock
	if err := alias.Of(s); err != nil {
		return err
	}
	*tb = alias
	return nil
}

type Job struct {
	Group       Group         `json:"-"`
	ProcTimeSec int64         `json:"-"`
	ConsumerID  uint16        `json:"-"`
	Operation   OperationType `json:"op"`
	TimeBlock   timeBlock     `json:"tb"`
	Key         string        `json:"key"`
}

func (opt OperationType) MarshalJSON() ([]byte, error) {
	if s, ok := operationTypeToString[opt]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown operation type %d", opt)
}

func (opt *OperationType) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := operationTypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown operation type %s", s)
	}
	*opt = v
	return nil
}

func (j *Job) String() string {
	bb := bytes.Buffer{}
	bb.WriteString(operationTypeToString[j.Operation])
	bb.WriteString(j.TimeBlock.String())
	bb.WriteString(j.Key)
	return bb.String()
}

func (j *Job) Of(s string) error {
	var alias Job
	op, ok := operationTypeFromString[s[:3]]
	if !ok {
		return fmt.Errorf("unknown operation type %s", s[:3])
	}
	alias.Operation = op
	if err := alias.TimeBlock.Of(s[3:14]); err != nil {
		return err
	}
	alias.Key = s[14:]
	*j = alias
	return nil
}

var delayFactorByTimeBlock = map[TimeBlockUnit]int64{
	TimeBlockUnitSecond: 1,
	TimeBlockUnitMinute: 1,
	TimeBlockUnitHour:   12,
	TimeBlockUnitDay:    24,
}

func (j *Job) DelaySec(base int64) int64 {
	return delayFactorByTimeBlock[j.TimeBlock.unit] * base
}

func GetGroups() []string {
	var groups []string
	for k := range GroupFromString {
		groups = append(groups, k)
	}
	return groups
}

// getProcTime calculate the timestamp in second when the job will be handled
func getProcTime(now, start time.Time, duration time.Duration, loc *time.Location) int64 {
	var (
		remaining time.Duration
	)

	if start.Add(duration).Before(now) {
		remaining = duration
	} else {
		durationMilli := duration.Milliseconds()
		remaining = time.Duration(durationMilli-(start.UnixMilli()%durationMilli)) * time.Millisecond
	}

	denominator := int64(duration.Seconds())
	if duration < time.Minute {
		denominator = 1
		return (now.Add(remaining).Unix() / denominator) * denominator
	}

	lt := getLocalTime(now, duration, loc)

	return lt.Add(duration).Unix()
}

func getLocalTime(t time.Time, d time.Duration, loc *time.Location) time.Time {
	lt := t.In(loc)

	year, month, day := lt.Date()
	if d >= 24*time.Hour {
		lt = time.Date(year, month, day, 0, 0, 0, 0, loc)
		return lt
	}

	hour, min, _ := lt.Clock()
	if d >= time.Hour {
		lt = time.Date(year, month, day, hour, 0, 0, 0, loc)
		return lt
	}

	lt = time.Date(year, month, day, hour, min, 0, 0, loc)

	return lt
}

func rollup1MinuteJob(key string, now, start time.Time, loc *time.Location, maxInstances uint16) Job {
	return Job{
		Group:       GroupRollup,
		ProcTimeSec: getProcTime(now, start, time.Minute, loc),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   opTimeSeriesRollup,
		TimeBlock:   toTimeBlock(start, TimeBlockUnitMinute, loc),
		Key:         key,
	}
}

func rollup1HourJob(key string, now, start time.Time, loc *time.Location, maxInstances uint16) Job {
	return Job{
		Group:       GroupRollup,
		ProcTimeSec: getProcTime(now, start, time.Hour, loc),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   opTimeSeriesRollup,
		TimeBlock:   toTimeBlock(start, TimeBlockUnitHour, loc),
		Key:         key,
	}
}

func rollup1DayJob(key string, now, start time.Time, loc *time.Location, maxInstances uint16) Job {
	return Job{
		Group:       GroupRollup,
		ProcTimeSec: getProcTime(now, start, 24*time.Hour, loc),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   opTimeSeriesRollup,
		TimeBlock:   toTimeBlock(start, TimeBlockUnitDay, loc),
		Key:         key,
	}
}

func deleteThingJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpTimeSeriesDelete,
		Key:         key,
	}
}

func deleteEventTypeJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpEventTypeDelete,
		Key:         key,
	}
}

func deleteCommandTypeJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpCommandTypeDelete,
		Key:         key,
	}
}

func deleteThingEventJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpThingEventDelete,
		Key:         key,
	}
}

func deleteThingCommandJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpThingCommandDelete,
		Key:         key,
	}
}

func deleteThingActionJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpThingActionDelete,
		Key:         key,
	}
}

func deleteThingTimeSeriesJob(key string, now time.Time, maxInstances uint16) Job {
	return Job{
		Group:       GroupDelete,
		ProcTimeSec: now.Unix(),
		ConsumerID:  consumerID(key, now, maxInstances),
		Operation:   OpThingTimeSeriesDelete,
		Key:         key,
	}
}

// hash https://github.com/SimonWaldherr/golang-benchmarks#hash
func consumerID(key string, now time.Time, maxInstances uint16) uint16 {
	bs := bytes.Buffer{}
	bs.WriteString(key)
	bs.WriteString(strconv.Itoa(int(now.Unix())))

	hash := crc32.NewIEEE()
	_, _ = hash.Write(bs.Bytes())
	return uint16(hash.Sum32()) % maxInstances
}
