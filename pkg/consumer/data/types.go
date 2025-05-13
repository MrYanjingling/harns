package data

import "time"

type RollupNumericItem struct {
	FirstTime    time.Time   `json:"F"`
	FirstValue   interface{} `json:"f"`
	LastTime     time.Time   `json:"L"`
	LastValue    interface{} `json:"l"`
	MinTime      time.Time   `json:"I"`
	MinValue     interface{} `json:"i"`
	MaxTime      time.Time   `json:"M"`
	MaxValue     interface{} `json:"m"`
	Avg          float64     `json:"a"`
	Sum          float64     `json:"s"`
	SD           float64     `json:"d"`
	GoodCnt      int64       `json:"g"`
	BadCnt       int64       `json:"b"`
	UncertainCnt int64       `json:"u"`
}

type RollupBoolItem struct {
	FirstTrueTime  time.Time `json:"F"`
	FirstFalseTime time.Time `json:"f"`
	LastTrueTime   time.Time `json:"L"`
	LastFalseTime  time.Time `json:"l"`
	LastValue      bool      `json:"v"`
	TrueCnt        int64     `json:"t"`
	TrueDuration   int64     `json:"T"` // milli second
	FalseCnt       int64     `json:"b"`
	FalseDuration  int64     `json:"B"` // milli second
	startTime      time.Time
	endTime        time.Time
}

type field struct {
	value interface{}
	time  time.Time
}
