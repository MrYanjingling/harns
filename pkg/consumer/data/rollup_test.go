package data

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"lightiot/pkg/logstorage/data"
	"reflect"
	"testing"
	"time"
)

var (
	loc, _ = time.LoadLocation("UTC")

	rawData1 = []field{
		{7.7, time.Date(2021, 11, 14, 21, 32, 4, 0, loc)},
		{6.6, time.Date(2021, 11, 14, 21, 32, 5, 0, loc)},
		{5.5, time.Date(2021, 11, 14, 21, 32, 6, 0, loc)},
		{4, time.Date(2021, 11, 14, 21, 32, 7, 0, loc)},
	}

	rollupRawData1 = &RollupNumericItem{
		FirstTime:    time.Date(2021, 11, 14, 21, 32, 4, 0, loc),
		FirstValue:   7.7,
		LastTime:     time.Date(2021, 11, 14, 21, 32, 7, 0, loc),
		LastValue:    4,
		MinTime:      time.Date(2021, 11, 14, 21, 32, 7, 0, loc),
		MinValue:     4,
		MaxTime:      time.Date(2021, 11, 14, 21, 32, 4, 0, loc),
		MaxValue:     7.7,
		Avg:          5.95,
		Sum:          23.8,
		SD:           1.368,
		GoodCnt:      4,
		BadCnt:       0,
		UncertainCnt: 0,
	}

	rawData2 = []field{
		{1.1, time.Date(2021, 11, 14, 21, 33, 1, 0, loc)},
		{2.2, time.Date(2021, 11, 14, 21, 33, 2, 0, loc)},
		{3.3, time.Date(2021, 11, 14, 21, 33, 3, 0, loc)},
	}

	rollupRawData2 = &RollupNumericItem{
		FirstTime:    time.Date(2021, 11, 14, 21, 33, 1, 0, loc),
		FirstValue:   1.1,
		LastTime:     time.Date(2021, 11, 14, 21, 33, 3, 0, loc),
		LastValue:    3.3,
		MinTime:      time.Date(2021, 11, 14, 21, 33, 1, 0, loc),
		MinValue:     1.1,
		MaxTime:      time.Date(2021, 11, 14, 21, 33, 3, 0, loc),
		MaxValue:     3.3,
		Avg:          2.2,
		Sum:          6.6,
		SD:           0.898,
		GoodCnt:      3,
		BadCnt:       0,
		UncertainCnt: 0,
	}

	rawData3 = []field{
		{12, time.Date(2021, 11, 14, 22, 33, 10, 0, loc)},
		{14.5, time.Date(2021, 11, 14, 22, 33, 11, 0, loc)},
		{100, time.Date(2021, 11, 14, 22, 33, 12, 0, loc)},
		{0.1, time.Date(2021, 11, 14, 22, 33, 12, 889000000, loc)},
		{16, time.Date(2021, 11, 14, 22, 33, 13, 0, loc)},
	}

	rollupRawData3 = &RollupNumericItem{
		FirstTime:    time.Date(2021, 11, 14, 22, 33, 10, 0, loc),
		FirstValue:   12,
		LastTime:     time.Date(2021, 11, 14, 22, 33, 13, 0, loc),
		LastValue:    16,
		MinTime:      time.Date(2021, 11, 14, 22, 33, 12, 889000000, loc),
		MinValue:     0.1,
		MaxTime:      time.Date(2021, 11, 14, 22, 33, 12, 0, loc),
		MaxValue:     100,
		Avg:          28.52,
		Sum:          142.6,
		SD:           36.175,
		GoodCnt:      5,
		BadCnt:       0,
		UncertainCnt: 0,
	}
)

func TestAggregateNumeric(t *testing.T) {
	tests := []struct {
		in       []field
		expected *RollupNumericItem
	}{
		{rawData1, rollupRawData1},
		{rawData2, rollupRawData2},
		{rawData3, rollupRawData3},
	}

	for i, tt := range tests {
		testName := fmt.Sprintf("Aggregate number %d", i+1)
		t.Run(testName, func(t *testing.T) {
			actual := aggregateNumeric(tt.in)
			equalsRollupNumericItem(t, tt.expected, actual)
		})
	}

}

func TestRollupNumeric(t *testing.T) {
	b1, _ := json.Marshal(rollupRawData1)
	rc1 := data.ResultColumn{
		Value: string(b1),
		Time:  time.Date(2021, 11, 14, 21, 33, 0, 0, loc),
	}

	b2, _ := json.Marshal(rollupRawData2)
	rc2 := data.ResultColumn{
		Value: string(b2),
		Time:  time.Date(2021, 11, 14, 21, 34, 0, 0, loc),
	}

	b3, _ := json.Marshal(rollupRawData3)
	rc3 := data.ResultColumn{
		Value: string(b3),
		Time:  time.Date(2021, 11, 14, 22, 34, 0, 0, loc),
	}

	tests := []struct {
		in       []data.ResultColumn
		expected *RollupNumericItem
	}{
		{[]data.ResultColumn{rc1, rc2}, &RollupNumericItem{
			FirstTime:    time.Date(2021, 11, 14, 21, 32, 4, 0, loc),
			FirstValue:   7.7,
			LastTime:     time.Date(2021, 11, 14, 21, 33, 3, 0, loc),
			LastValue:    3.3,
			MinTime:      time.Date(2021, 11, 14, 21, 33, 1, 0, loc),
			MinValue:     1.1,
			MaxTime:      time.Date(2021, 11, 14, 21, 32, 4, 0, loc),
			MaxValue:     7.7,
			Avg:          4.343,
			Sum:          30.4,
			SD:           2.204,
			GoodCnt:      7,
			BadCnt:       0,
			UncertainCnt: 0,
		}},
		{[]data.ResultColumn{rc2, rc3}, &RollupNumericItem{
			FirstTime:    time.Date(2021, 11, 14, 21, 33, 1, 0, loc),
			FirstValue:   1.1,
			LastTime:     time.Date(2021, 11, 14, 22, 33, 13, 0, loc),
			LastValue:    16.0,
			MinTime:      time.Date(2021, 11, 14, 22, 33, 12, 889000000, loc),
			MinValue:     0.1,
			MaxTime:      time.Date(2021, 11, 14, 22, 33, 12, 0, loc),
			MaxValue:     100.0,
			Avg:          18.65,
			Sum:          149.2,
			SD:           31.314,
			GoodCnt:      8,
			BadCnt:       0,
			UncertainCnt: 0,
		}},
		{[]data.ResultColumn{rc1, rc2, rc3}, &RollupNumericItem{
			FirstTime:    time.Date(2021, 11, 14, 21, 32, 4, 0, loc),
			FirstValue:   7.7,
			LastTime:     time.Date(2021, 11, 14, 22, 33, 13, 0, loc),
			LastValue:    16.0,
			MinTime:      time.Date(2021, 11, 14, 22, 33, 12, 889000000, loc),
			MinValue:     0.1,
			MaxTime:      time.Date(2021, 11, 14, 22, 33, 12, 0, loc),
			MaxValue:     100.0,
			Avg:          14.417,
			Sum:          173,
			SD:           26.271,
			GoodCnt:      12,
			BadCnt:       0,
			UncertainCnt: 0,
		}},
	}

	for i, tt := range tests {
		testName := fmt.Sprintf("Rollup number %d", i+1)
		t.Run(testName, func(t *testing.T) {
			actual := rollupNumeric(tt.in)
			equalsRollupNumericItem(t, tt.expected, actual)
		})
	}
}

// equalsRollupNumericItem is able to compare float
func equalsRollupNumericItem(t assert.TestingT, expected, actual *RollupNumericItem) {
	assert.Equal(t, expected.FirstTime, actual.FirstTime, "FirstTime")
	assert.Equal(t, expected.FirstValue, actual.FirstValue, "FirstValue")
	assert.Equal(t, expected.LastTime, actual.LastTime, "LastTime")
	assert.Equal(t, expected.LastValue, actual.LastValue, "LastValue")
	assert.Equal(t, expected.MinTime, actual.MinTime, "MinTime")
	assert.Equal(t, expected.MinValue, actual.MinValue, "MinValue")
	assert.Equal(t, expected.MaxTime, actual.MaxTime, "MaxTime")
	assert.Equal(t, expected.MaxValue, actual.MaxValue, "MaxValue")
	assert.InDelta(t, expected.Avg, actual.Avg, 0.001, "Avg")
	assert.InDelta(t, expected.Sum, actual.Sum, 0.001, "Sum")
	assert.InDelta(t, expected.SD, actual.SD, 0.001, "SD")
	assert.Equal(t, expected.GoodCnt, actual.GoodCnt, "GoodCnt")
	assert.Equal(t, expected.BadCnt, actual.BadCnt, "BadCnt")
	assert.Equal(t, expected.UncertainCnt, actual.UncertainCnt, "UncertainCnt")
}

var (
	rawBool1 = []field{
		{true, time.Date(2021, 11, 15, 13, 51, 4, 0, loc)},
	}
	rollupBool1 = &RollupBoolItem{
		FirstTrueTime:  time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
		FirstFalseTime: time.Time{},
		LastTrueTime:   time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
		LastFalseTime:  time.Time{},
		LastValue:      true,
		TrueCnt:        1,
		TrueDuration:   60 * 1000,
		FalseCnt:       0,
		FalseDuration:  0,
	}

	rawBool2 = []field{
		{true, time.Date(2021, 11, 15, 13, 52, 4, 0, loc)},
		{false, time.Date(2021, 11, 15, 13, 52, 20, 0, loc)},
	}
	rollupBool2 = &RollupBoolItem{
		FirstTrueTime:  time.Date(2021, 11, 15, 13, 52, 4, 0, loc),
		FirstFalseTime: time.Date(2021, 11, 15, 13, 52, 20, 0, loc),
		LastTrueTime:   time.Date(2021, 11, 15, 13, 52, 4, 0, loc),
		LastFalseTime:  time.Date(2021, 11, 15, 13, 52, 20, 0, loc),
		LastValue:      false,
		TrueCnt:        1,
		TrueDuration:   20 * 1000,
		FalseCnt:       1,
		FalseDuration:  40 * 1000,
	}

	rawBool3 = []field{
		{true, time.Date(2021, 11, 15, 13, 54, 4, 0, loc)},
		{true, time.Date(2021, 11, 15, 13, 54, 6, 0, loc)},
		{false, time.Date(2021, 11, 15, 13, 54, 20, 0, loc)},
	}
	rollupBool3 = &RollupBoolItem{
		FirstTrueTime:  time.Date(2021, 11, 15, 13, 54, 4, 0, loc),
		FirstFalseTime: time.Date(2021, 11, 15, 13, 54, 20, 0, loc),
		LastTrueTime:   time.Date(2021, 11, 15, 13, 54, 6, 0, loc),
		LastFalseTime:  time.Date(2021, 11, 15, 13, 54, 20, 0, loc),
		LastValue:      false,
		TrueCnt:        2,
		TrueDuration:   20 * 1000,
		FalseCnt:       1,
		FalseDuration:  40 * 1000,
	}

	rawBool4 = []field{
		{true, time.Date(2021, 11, 15, 13, 55, 4, 0, loc)},
		{true, time.Date(2021, 11, 15, 13, 55, 6, 0, loc)},
		{true, time.Date(2021, 11, 15, 13, 55, 8, 0, loc)},
		{false, time.Date(2021, 11, 15, 13, 55, 20, 0, loc)},
		{true, time.Date(2021, 11, 15, 13, 55, 30, 0, loc)},
		{false, time.Date(2021, 11, 15, 13, 55, 32, 0, loc)},
	}
	rollupBool4 = &RollupBoolItem{
		FirstTrueTime:  time.Date(2021, 11, 15, 13, 55, 4, 0, loc),
		FirstFalseTime: time.Date(2021, 11, 15, 13, 55, 20, 0, loc),
		LastTrueTime:   time.Date(2021, 11, 15, 13, 55, 30, 0, loc),
		LastFalseTime:  time.Date(2021, 11, 15, 13, 55, 32, 0, loc),
		LastValue:      false,
		TrueCnt:        4,
		TrueDuration:   22 * 1000,
		FalseCnt:       2,
		FalseDuration:  38 * 1000,
	}

	rawBool5 = []field{
		{true, time.Date(2021, 11, 15, 13, 51, 4, 0, loc)},
		{true, time.Date(2021, 11, 15, 13, 51, 6, 0, loc)},
		{true, time.Date(2021, 11, 15, 13, 51, 8, 0, loc)},
	}
	rollupBool5 = &RollupBoolItem{
		FirstTrueTime:  time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
		FirstFalseTime: time.Time{},
		LastTrueTime:   time.Date(2021, 11, 15, 13, 51, 8, 0, loc),
		LastFalseTime:  time.Time{},
		LastValue:      true,
		TrueCnt:        3,
		TrueDuration:   60 * 1000,
		FalseCnt:       0,
		FalseDuration:  0,
	}

	rawBool6 = []field{
		{false, time.Date(2021, 12, 22, 03, 02, 19, 116000000, loc)},
		{false, time.Date(2021, 12, 22, 03, 02, 29, 17000000, loc)},
		{false, time.Date(2021, 12, 22, 03, 02, 39, 182000000, loc)},
		{false, time.Date(2021, 12, 22, 03, 02, 49, 19000000, loc)},
		{false, time.Date(2021, 12, 22, 03, 02, 59, 201000000, loc)},
	}
	rollupBool6 = &RollupBoolItem{
		FirstTrueTime:  time.Time{},
		FirstFalseTime: time.Date(2021, 12, 22, 03, 02, 19, 116000000, loc),
		LastTrueTime:   time.Time{},
		LastFalseTime:  time.Date(2021, 12, 22, 03, 02, 59, 201000000, loc),
		LastValue:      false,
		TrueCnt:        0,
		TrueDuration:   0,
		FalseCnt:       5,
		FalseDuration:  60 * 1000,
	}

	rawBool7 = []field{
		{false, time.Date(2021, 12, 27, 00, 00, 00, 132000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 03, 539000000, loc)},
		{false, time.Date(2021, 12, 27, 00, 00, 16, 596000000, loc)},
		{false, time.Date(2021, 12, 27, 00, 00, 22, 766000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 27, 411000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 28, 763000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 31, 758000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 35, 303000000, loc)},
		{false, time.Date(2021, 12, 27, 00, 00, 38, 916000000, loc)},
		{false, time.Date(2021, 12, 27, 00, 00, 44, 413000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 46, 512000000, loc)},
		{false, time.Date(2021, 12, 27, 00, 00, 53, 241000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 54, 190000000, loc)},
		{true, time.Date(2021, 12, 27, 00, 00, 58, 388000000, loc)},
	}
	rollupBool7 = &RollupBoolItem{
		FirstTrueTime:  time.Date(2021, 12, 27, 00, 00, 03, 539000000, loc),
		FirstFalseTime: time.Date(2021, 12, 27, 00, 00, 00, 132000000, loc),
		LastTrueTime:   time.Date(2021, 12, 27, 00, 00, 58, 388000000, loc),
		LastFalseTime:  time.Date(2021, 12, 27, 00, 00, 53, 241000000, loc),
		LastValue:      true,
		TrueCnt:        8,
		TrueDuration:   37101,
		FalseCnt:       6,
		FalseDuration:  22899,
	}
)

func TestAggregateBool(t *testing.T) {
	tests := []struct {
		in       []field
		expected *RollupBoolItem
	}{
		{rawBool1, rollupBool1},
		{rawBool2, rollupBool2},
		{rawBool3, rollupBool3},
		{rawBool4, rollupBool4},
		{rawBool5, rollupBool5},
		{rawBool6, rollupBool6},
		{rawBool7, rollupBool7},
	}
	for i, tt := range tests {
		testName := fmt.Sprintf("Aggregate bool %d", i+1)
		start := tt.in[0].time.Truncate(time.Minute)
		end := start.Add(time.Minute)
		t.Run(testName, func(t *testing.T) {
			actual := aggregateBool(tt.in, start, end)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestRollupBool(t *testing.T) {
	b1, _ := json.Marshal(rollupBool1)
	rc1 := data.ResultColumn{
		Value: string(b1),
		Time:  time.Date(2021, 11, 15, 13, 52, 0, 0, loc),
	}

	b2, _ := json.Marshal(rollupBool2)
	rc2 := data.ResultColumn{
		Value: string(b2),
		Time:  time.Date(2021, 11, 15, 13, 53, 0, 0, loc),
	}

	b3, _ := json.Marshal(rollupBool3)
	rc3 := data.ResultColumn{
		Value: string(b3),
		Time:  time.Date(2021, 11, 15, 13, 55, 0, 0, loc),
	}

	b4, _ := json.Marshal(rollupBool4)
	rc4 := data.ResultColumn{
		Value: string(b4),
		Time:  time.Date(2021, 11, 15, 13, 56, 0, 0, loc),
	}

	tests := []struct {
		in       []data.ResultColumn
		expected *RollupBoolItem
	}{
		{[]data.ResultColumn{rc1}, &RollupBoolItem{
			FirstTrueTime:  time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
			FirstFalseTime: time.Time{},
			LastTrueTime:   time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
			LastFalseTime:  time.Time{},
			LastValue:      true,
			TrueCnt:        1,
			TrueDuration:   3600 * 1000,
			FalseCnt:       0,
			FalseDuration:  0,
		}},
		{[]data.ResultColumn{rc1, rc2}, &RollupBoolItem{
			FirstTrueTime:  time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
			FirstFalseTime: time.Date(2021, 11, 15, 13, 52, 20, 0, loc),
			LastTrueTime:   time.Date(2021, 11, 15, 13, 52, 4, 0, loc),
			LastFalseTime:  time.Date(2021, 11, 15, 13, 52, 20, 0, loc),
			LastValue:      false,
			TrueCnt:        2,
			TrueDuration:   (51*60 + 80) * 1000,
			FalseCnt:       1,
			FalseDuration:  (7*60 + 40) * 1000,
		}},
		{[]data.ResultColumn{rc2, rc3}, &RollupBoolItem{
			FirstTrueTime:  time.Date(2021, 11, 15, 13, 52, 4, 0, loc),
			FirstFalseTime: time.Date(2021, 11, 15, 13, 52, 20, 0, loc),
			LastTrueTime:   time.Date(2021, 11, 15, 13, 54, 6, 0, loc),
			LastFalseTime:  time.Date(2021, 11, 15, 13, 54, 20, 0, loc),
			LastValue:      false,
			TrueCnt:        3,
			TrueDuration:   (52*60 + 20 + 16) * 1000,
			FalseCnt:       2,
			FalseDuration:  (1*60 + 44 + 5*60 + 40) * 1000,
		}},
		{[]data.ResultColumn{rc1, rc2, rc3, rc4}, &RollupBoolItem{
			FirstTrueTime:  time.Date(2021, 11, 15, 13, 51, 4, 0, loc),
			FirstFalseTime: time.Date(2021, 11, 15, 13, 52, 20, 0, loc),
			LastTrueTime:   time.Date(2021, 11, 15, 13, 55, 30, 0, loc),
			LastFalseTime:  time.Date(2021, 11, 15, 13, 55, 32, 0, loc),
			LastValue:      false,
			TrueCnt:        8,
			TrueDuration:   (52*60 + 20 + 16 + 16 + 2) * 1000,
			FalseCnt:       4,
			FalseDuration:  (1*60 + 44 + 44 + 10 + 4*60 + 28) * 1000,
		}},
	}

	for i, tt := range tests {
		testName := fmt.Sprintf("Rollup bool %d", i+1)
		t.Run(testName, func(t *testing.T) {
			actual := rollupBool(tt.in, time.Date(2021, 11, 15, 13, 00, 0, 0, loc), time.Date(2021, 11, 15, 14, 00, 0, 0, loc), int64(time.Minute/time.Millisecond))
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, int64(60*60*1000), actual.TrueDuration+actual.FalseDuration)
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		in       interface{}
		expected float64
	}{
		{10, float64(10)},
		{9234567899876, float64(9234567899876)},
		{int64(32), float64(32)},
		{15.6, float64(15.6)},
		{float32(15.6), float64(0)},
		{"abc", float64(0)},
	}
	for _, tt := range tests {
		testName := fmt.Sprintf("%s %v to float64", reflect.TypeOf(tt.in), tt.in)
		t.Run(testName, func(t *testing.T) {
			actual := toFloat64(tt.in)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
