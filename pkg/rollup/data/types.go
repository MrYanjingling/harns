package data

import "time"

type RollupNumericField byte

const (
	RollupNumericFieldFirstTime = iota
	RollupNumericFieldFirstValue
	RollupNumericFieldLastTime
	RollupNumericFieldLastValue
	RollupNumericFieldMinTime
	RollupNumericFieldMinValue
	RollupNumericFieldMaxTime
	RollupNumericFieldMaxValue
	RollupNumericFieldAvg
	RollupNumericFieldSum
	RollupNumericFieldSD
	RollupNumericFieldGoodCnt
	RollupNumericFieldBadCnt
	RollupNumericFieldUncertainCnt
)

var RollupNumericFieldFromString = map[string]RollupNumericField{
	"firsttime":    RollupNumericFieldFirstTime,
	"firstvalue":   RollupNumericFieldFirstValue,
	"lasttime":     RollupNumericFieldLastTime,
	"lastvalue":    RollupNumericFieldLastValue,
	"mintime":      RollupNumericFieldMinTime,
	"minvalue":     RollupNumericFieldMinValue,
	"maxtime":      RollupNumericFieldMaxTime,
	"maxvalue":     RollupNumericFieldMaxValue,
	"avg":          RollupNumericFieldAvg,
	"sum":          RollupNumericFieldSum,
	"sd":           RollupNumericFieldSD,
	"goodcnt":      RollupNumericFieldGoodCnt,
	"badcnt":       RollupNumericFieldBadCnt,
	"uncertaincnt": RollupNumericFieldUncertainCnt,
}

var RollupNumericFieldToString = map[RollupNumericField]string{
	RollupNumericFieldFirstTime:    "firstTime",
	RollupNumericFieldFirstValue:   "firstValue",
	RollupNumericFieldLastTime:     "lastTime",
	RollupNumericFieldLastValue:    "lastValue",
	RollupNumericFieldMinTime:      "minTime",
	RollupNumericFieldMinValue:     "minValue",
	RollupNumericFieldMaxTime:      "maxTime",
	RollupNumericFieldMaxValue:     "maxValue",
	RollupNumericFieldAvg:          "avg",
	RollupNumericFieldSum:          "sum",
	RollupNumericFieldSD:           "sd",
	RollupNumericFieldGoodCnt:      "goodCnt",
	RollupNumericFieldBadCnt:       "badCnt",
	RollupNumericFieldUncertainCnt: "uncertainCnt",
}

type RollupBoolField byte

const (
	RollupBoolFieldFirstTrueTime = iota
	RollupBoolFieldFirstFalseTime
	RollupBoolFieldLastTrueTime
	RollupBoolFieldLastFalseTime
	RollupBoolFieldLastValue
	RollupBoolFieldTrueCnt
	RollupBoolFieldTrueDuration
	RollupBoolFieldFalseCnt
	RollupBoolFieldFalseDuration
)

var RollupBoolFieldToString = map[RollupBoolField]string{
	RollupBoolFieldFirstTrueTime:  "firstTrueTime",
	RollupBoolFieldFirstFalseTime: "firstFalseTime",
	RollupBoolFieldLastTrueTime:   "lastTrueTime",
	RollupBoolFieldLastFalseTime:  "lastFalseTime",
	RollupBoolFieldLastValue:      "lastValue",
	RollupBoolFieldTrueCnt:        "trueCnt",
	RollupBoolFieldTrueDuration:   "trueDuration",
	RollupBoolFieldFalseCnt:       "falseCnt",
	RollupBoolFieldFalseDuration:  "falseDuration",
}

var RollupBoolFieldFromString = map[string]RollupBoolField{
	"firsttruetime":  RollupBoolFieldFirstTrueTime,
	"firstfalsetime": RollupBoolFieldFirstFalseTime,
	"lasttruetime":   RollupBoolFieldLastTrueTime,
	"lastfalsetime":  RollupBoolFieldLastFalseTime,
	"lastvalue":      RollupBoolFieldLastValue,
	"truecnt":        RollupBoolFieldTrueCnt,
	"trueduration":   RollupBoolFieldTrueDuration,
	"falsecnt":       RollupBoolFieldFalseCnt,
	"falseduration":  RollupBoolFieldFalseDuration,
}

type QueryIntervalUnit byte

const (
	QueryIntervalUnitSecond = 's'
	QueryIntervalUnitMinute = 'm'
	QueryIntervalUnitHour   = 'h'
	QueryIntervalUnitDay    = 'D'
	QueryIntervalUnitWeek   = 'W'
	QueryIntervalUnitMonth  = 'M'
	DaysOneWeek             = 7
)

var QueryIntervalUnitToDuration = map[QueryIntervalUnit]time.Duration{
	QueryIntervalUnitSecond: time.Second,
	QueryIntervalUnitMinute: time.Minute,
	QueryIntervalUnitHour:   time.Hour,
	QueryIntervalUnitDay:    24 * time.Hour,
	QueryIntervalUnitWeek:   7 * 24 * time.Hour,
	QueryIntervalUnitMonth:  0,
}

type Interval struct {
	amount int
	unit   QueryIntervalUnit
}

type RollupRecord struct {
	end  time.Time
	item map[string]interface{}
}

type Response struct {
	Rollups []map[string]interface{} `json:"rollups"`
}

type Request struct {
	thingId             string
	propertySetName     string
	start               string
	end                 string
	interval            string
	filter              string
	limit               string
	weekStartFromSunday bool
}
