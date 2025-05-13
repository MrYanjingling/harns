package job

import "time"

type Group byte

const (
	GroupRollup Group = iota
	GroupDelete
)

var (
	GroupToString = map[Group]string{
		GroupRollup: "rup",
		GroupDelete: "del",
	}

	GroupFromString = map[string]Group{
		"rup": GroupRollup,
		"del": GroupDelete,
	}
)

type OperationType byte

const (
	opTimeSeriesRollup OperationType = iota
	OpTimeSeriesDelete
	OpEventTypeDelete
	OpCommandTypeDelete
	OpThingCommandDelete
	OpThingActionDelete
	OpThingEventDelete
	OpThingTimeSeriesDelete
)

var (
	operationTypeToString = map[OperationType]string{
		opTimeSeriesRollup:      "TSR",
		OpTimeSeriesDelete:      "TSD",
		OpEventTypeDelete:       "ETD",
		OpCommandTypeDelete:     "CTD",
		OpThingCommandDelete:    "TCD",
		OpThingActionDelete:     "TAD",
		OpThingEventDelete:      "TED",
		OpThingTimeSeriesDelete: "TTD",
	}

	operationTypeFromString = map[string]OperationType{
		"TSR": opTimeSeriesRollup,
		"TSD": OpTimeSeriesDelete,
		"ETD": OpEventTypeDelete,
		"CTD": OpCommandTypeDelete,
		"TCD": OpThingCommandDelete,
		"TAD": OpThingActionDelete,
		"TED": OpThingEventDelete,
		"TTD": OpThingTimeSeriesDelete,
	}
)

type TimeBlockUnit byte

const (
	TimeBlockUnitSecond TimeBlockUnit = iota
	TimeBlockUnitMinute
	TimeBlockUnitHour
	TimeBlockUnitDay
	TimeBlockUnitWeek
	TimeBlockUnitMonth
)

var (
	TimeBlockUnitToString = map[TimeBlockUnit]byte{
		TimeBlockUnitSecond: 's',
		TimeBlockUnitMinute: 'm',
		TimeBlockUnitHour:   'h',
		TimeBlockUnitDay:    'd',
		TimeBlockUnitWeek:   'W',
		TimeBlockUnitMonth:  'M',
	}

	TimeBlockUnitFromString = map[byte]TimeBlockUnit{
		's': TimeBlockUnitSecond,
		'm': TimeBlockUnitMinute,
		'h': TimeBlockUnitHour,
		'd': TimeBlockUnitDay,
		'W': TimeBlockUnitWeek,
		'M': TimeBlockUnitMonth,
	}

	TimeBlockUnitDuration = map[TimeBlockUnit]int64{
		TimeBlockUnitSecond: int64(time.Second / time.Millisecond),
		TimeBlockUnitMinute: int64(time.Minute / time.Millisecond),
		TimeBlockUnitHour:   int64(time.Hour / time.Millisecond),
		TimeBlockUnitDay:    int64(24 * time.Hour / time.Millisecond),
		TimeBlockUnitWeek:   int64(7 * 24 * time.Hour / time.Millisecond),
		TimeBlockUnitMonth:  int64(0),
	}

	DaysPerMonth = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
)

const (
	jobDBName      = "jobs"
	jobWorkerID    = "id"
	jobProcTime    = "_time"
	jobData        = "data"
	jobPlaceHolder = "pl"
	jobGroup       = "jr"
)
