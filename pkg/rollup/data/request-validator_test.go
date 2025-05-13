package data

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_alignStartDate(t *testing.T) {
	var tests = []struct {
		timeZone  string
		startDate string
		interval  int
		unit      QueryIntervalUnit
		expected  string
	}{
		// 1) 1s
		{"Asia/Chongqing", "2021-11-03T00:02:10Z", 1, 's', "\"2021-11-03T08:02:10+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 's', "\"2021-11-03T08:02:10+08:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10Z", 1, 's', "\"2021-11-02T17:02:10-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 's', "\"2021-11-02T17:02:10-07:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10Z", 1, 's', "\"2021-11-03T05:32:10+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 's', "\"2021-11-03T05:32:10+05:30\""},

		// 2) 1m
		// normal
		{"Asia/Chongqing", "2021-11-03T00:02:00Z", 1, 'm', "\"2021-11-03T08:02:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 'm', "\"2021-11-03T08:02:00+08:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:00Z", 1, 'm', "\"2021-11-03T05:32:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 'm', "\"2021-11-03T05:32:00+05:30\""},
		// cross day
		{"America/Los_Angeles", "2021-11-03T00:02:00Z", 1, 'm', "\"2021-11-02T17:02:00-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 'm', "\"2021-11-02T17:02:00-07:00\""},
		{"Asia/Kolkata", "2021-11-03T20:02:00Z", 1, 'm', "\"2021-11-04T01:32:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:10.001Z", 1, 'm', "\"2021-11-04T01:32:00+05:30\""},
		// cross mouth
		{"America/Los_Angeles", "2021-11-01T02:02:00Z", 1, 'm', "\"2021-10-31T19:02:00-07:00\""},
		{"America/Los_Angeles", "2021-11-01T02:02:10.001Z", 1, 'm', "\"2021-10-31T19:02:00-07:00\""},
		{"Asia/Kolkata", "2021-10-31T23:02:00Z", 1, 'm', "\"2021-11-01T04:32:00+05:30\""},
		{"Asia/Kolkata", "2021-10-31T23:02:10.001Z", 1, 'm', "\"2021-11-01T04:32:00+05:30\""},
		// cross year
		{"America/Los_Angeles", "2022-01-01T02:02:00Z", 1, 'm', "\"2021-12-31T18:02:00-08:00\""},
		{"America/Los_Angeles", "2022-01-01T02:02:10.001Z", 1, 'm', "\"2021-12-31T18:02:00-08:00\""},
		{"Asia/Kolkata", "2021-12-31T23:02:00Z", 1, 'm', "\"2022-01-01T04:32:00+05:30\""},
		{"Asia/Kolkata", "2021-12-31T23:02:10.001Z", 1, 'm', "\"2022-01-01T04:32:00+05:30\""},

		// 3) 1h
		// normal
		{"Asia/Chongqing", "2021-11-03T00:02:00Z", 1, 'h', "\"2021-11-03T08:00:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 'h', "\"2021-11-03T08:00:00+08:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:00Z", 1, 'h', "\"2021-11-03T05:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 'h', "\"2021-11-03T05:00:00+05:30\""},
		// cross day
		{"America/Los_Angeles", "2021-11-03T00:02:00Z", 1, 'h', "\"2021-11-02T17:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 'h', "\"2021-11-02T17:00:00-07:00\""},
		{"Asia/Kolkata", "2021-11-03T20:02:00Z", 1, 'h', "\"2021-11-04T01:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:10.001Z", 1, 'h', "\"2021-11-04T01:00:00+05:30\""},
		// cross mouth
		{"America/Los_Angeles", "2021-11-01T02:02:00Z", 1, 'h', "\"2021-10-31T19:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-01T02:02:10.001Z", 1, 'h', "\"2021-10-31T19:00:00-07:00\""},
		{"Asia/Kolkata", "2021-10-31T23:02:00Z", 1, 'h', "\"2021-11-01T04:00:00+05:30\""},
		{"Asia/Kolkata", "2021-10-31T23:02:10.001Z", 1, 'h', "\"2021-11-01T04:00:00+05:30\""},
		// cross year
		{"America/Los_Angeles", "2022-01-01T02:02:00Z", 1, 'h', "\"2021-12-31T18:00:00-08:00\""},
		{"America/Los_Angeles", "2022-01-01T02:02:10.001Z", 1, 'h', "\"2021-12-31T18:00:00-08:00\""},
		{"Asia/Kolkata", "2021-12-31T23:02:00Z", 1, 'h', "\"2022-01-01T04:00:00+05:30\""},
		{"Asia/Kolkata", "2021-12-31T23:02:10.001Z", 1, 'h', "\"2022-01-01T04:00:00+05:30\""},

		// 4) 1D
		// normal
		{"Asia/Chongqing", "2021-11-03T00:02:00Z", 1, 'D', "\"2021-11-03T00:00:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 'D', "\"2021-11-03T00:00:00+08:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:00Z", 1, 'D', "\"2021-11-03T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 'D', "\"2021-11-03T00:00:00+05:30\""},
		// cross day
		{"America/Los_Angeles", "2021-11-03T00:02:00Z", 1, 'D', "\"2021-11-02T00:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 'D', "\"2021-11-02T00:00:00-07:00\""},
		{"Asia/Kolkata", "2021-11-03T20:02:00Z", 1, 'D', "\"2021-11-04T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:10.001Z", 1, 'D', "\"2021-11-04T00:00:00+05:30\""},
		// cross mouth
		{"America/Los_Angeles", "2021-11-01T02:02:00Z", 1, 'D', "\"2021-10-31T00:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-01T02:02:10.001Z", 1, 'D', "\"2021-10-31T00:00:00-07:00\""},
		{"Asia/Kolkata", "2021-10-31T23:02:00Z", 1, 'D', "\"2021-11-01T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-10-31T23:02:10.001Z", 1, 'D', "\"2021-11-01T00:00:00+05:30\""},
		// cross year
		{"America/Los_Angeles", "2022-01-01T02:02:00Z", 1, 'D', "\"2021-12-31T00:00:00-08:00\""},
		{"America/Los_Angeles", "2022-01-01T02:02:10.001Z", 1, 'D', "\"2021-12-31T00:00:00-08:00\""},
		{"Asia/Kolkata", "2021-12-31T23:02:00Z", 1, 'D', "\"2022-01-01T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-12-31T23:02:10.001Z", 1, 'D', "\"2022-01-01T00:00:00+05:30\""},
	}
	for _, tt := range tests {
		start, _ := time.Parse(time.RFC3339Nano, tt.startDate)
		location, _ := time.LoadLocation(tt.timeZone)
		start = start.UTC().In(location)
		testName := "Test align start date:" + tt.startDate + ", time zone:" + tt.timeZone + ", interval: 1" + string(tt.unit)
		validator := &RequestValidator{
			startDate: start,
			interval:  Interval{tt.interval, tt.unit},
			location:  location,
		}
		t.Run(testName, func(t *testing.T) {
			validator.alignStartDate()
			tmpDate, _ := json.Marshal(validator.startDate)
			actual := string(tmpDate)
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("actual %v, expect %v", actual, tt.expected)
			}
		})
	}
}

func Test_alignEndDate(t *testing.T) {
	var tests = []struct {
		timeZone string
		endDate  string
		interval int
		unit     QueryIntervalUnit
		expected string
	}{
		// 1) 1s
		{"Asia/Chongqing", "2021-11-03T00:02:10Z", 1, 's', "\"2021-11-03T08:02:10+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 's', "\"2021-11-03T08:02:10+08:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10Z", 1, 's', "\"2021-11-02T17:02:10-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 's', "\"2021-11-02T17:02:10-07:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10Z", 1, 's', "\"2021-11-03T05:32:10+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 's', "\"2021-11-03T05:32:10+05:30\""},

		// 2) 1m
		// normal
		{"Asia/Chongqing", "2021-11-03T00:02:00Z", 1, 'm', "\"2021-11-03T08:02:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 'm', "\"2021-11-03T08:03:00+08:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:00Z", 1, 'm', "\"2021-11-03T05:32:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 'm', "\"2021-11-03T05:33:00+05:30\""},
		// cross day
		{"America/Los_Angeles", "2021-11-03T00:02:00Z", 1, 'm', "\"2021-11-02T17:02:00-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 'm', "\"2021-11-02T17:03:00-07:00\""},
		{"Asia/Kolkata", "2021-11-03T20:02:00Z", 1, 'm', "\"2021-11-04T01:32:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:10.001Z", 1, 'm', "\"2021-11-04T01:33:00+05:30\""},
		// cross mouth
		{"America/Los_Angeles", "2021-11-01T02:02:00Z", 1, 'm', "\"2021-10-31T19:02:00-07:00\""},
		{"America/Los_Angeles", "2021-11-01T02:02:10.001Z", 1, 'm', "\"2021-10-31T19:03:00-07:00\""},
		{"Asia/Kolkata", "2021-10-31T23:02:00Z", 1, 'm', "\"2021-11-01T04:32:00+05:30\""},
		{"Asia/Kolkata", "2021-10-31T23:02:10.001Z", 1, 'm', "\"2021-11-01T04:33:00+05:30\""},
		// cross year
		{"America/Los_Angeles", "2022-01-01T02:02:00Z", 1, 'm', "\"2021-12-31T18:02:00-08:00\""},
		{"America/Los_Angeles", "2022-01-01T02:02:10.001Z", 1, 'm', "\"2021-12-31T18:03:00-08:00\""},
		{"Asia/Kolkata", "2021-12-31T23:02:00Z", 1, 'm', "\"2022-01-01T04:32:00+05:30\""},
		{"Asia/Kolkata", "2021-12-31T23:02:10.001Z", 1, 'm', "\"2022-01-01T04:33:00+05:30\""},

		// 3) 1h
		// normal
		{"Asia/Chongqing", "2021-11-03T00:00:00Z", 1, 'h', "\"2021-11-03T08:00:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:00Z", 1, 'h', "\"2021-11-03T09:00:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 'h', "\"2021-11-03T09:00:00+08:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:00Z", 1, 'h', "\"2021-11-03T06:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 'h', "\"2021-11-03T06:00:00+05:30\""},
		// cross day
		{"America/Los_Angeles", "2021-11-03T00:02:00Z", 1, 'h', "\"2021-11-02T18:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 'h', "\"2021-11-02T18:00:00-07:00\""},
		{"Asia/Kolkata", "2021-11-03T20:00:00Z", 1, 'h', "\"2021-11-04T02:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:31:00Z", 1, 'h', "\"2021-11-04T03:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:00Z", 1, 'h', "\"2021-11-04T02:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:10.001Z", 1, 'h', "\"2021-11-04T02:00:00+05:30\""},
		// cross mouth
		{"America/Los_Angeles", "2021-11-01T02:02:00Z", 1, 'h', "\"2021-10-31T20:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-01T02:02:10.001Z", 1, 'h', "\"2021-10-31T20:00:00-07:00\""},
		{"Asia/Kolkata", "2021-10-31T23:02:00Z", 1, 'h', "\"2021-11-01T05:00:00+05:30\""},
		{"Asia/Kolkata", "2021-10-31T23:02:10.001Z", 1, 'h', "\"2021-11-01T05:00:00+05:30\""},
		// cross year
		{"America/Los_Angeles", "2022-01-01T02:02:00Z", 1, 'h', "\"2021-12-31T19:00:00-08:00\""},
		{"America/Los_Angeles", "2022-01-01T02:02:10.001Z", 1, 'h', "\"2021-12-31T19:00:00-08:00\""},
		{"Asia/Kolkata", "2021-12-31T23:02:00Z", 1, 'h', "\"2022-01-01T05:00:00+05:30\""},
		{"Asia/Kolkata", "2021-12-31T23:02:10.001Z", 1, 'h', "\"2022-01-01T05:00:00+05:30\""},

		// 4) 1D
		// normal
		{"Asia/Chongqing", "2021-11-03T00:02:00Z", 1, 'D', "\"2021-11-04T00:00:00+08:00\""},
		{"Asia/Chongqing", "2021-11-03T00:02:10.001Z", 1, 'D', "\"2021-11-04T00:00:00+08:00\""},
		{"Asia/Kolkata", "2021-11-03T00:02:00Z", 1, 'D', "\"2021-11-04T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T00:02:10.001Z", 1, 'D', "\"2021-11-04T00:00:00+05:30\""},
		// cross day
		{"America/Los_Angeles", "2021-11-03T00:02:00Z", 1, 'D', "\"2021-11-03T00:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-03T00:02:10.001Z", 1, 'D', "\"2021-11-03T00:00:00-07:00\""},
		{"Asia/Kolkata", "2021-11-03T20:02:00Z", 1, 'D', "\"2021-11-05T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-11-03T20:02:10.001Z", 1, 'D', "\"2021-11-05T00:00:00+05:30\""},
		// cross mouth
		{"America/Los_Angeles", "2021-11-01T02:02:00Z", 1, 'D', "\"2021-11-01T00:00:00-07:00\""},
		{"America/Los_Angeles", "2021-11-01T02:02:10.001Z", 1, 'D', "\"2021-11-01T00:00:00-07:00\""},
		{"Asia/Kolkata", "2021-10-31T23:02:00Z", 1, 'D', "\"2021-11-02T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-10-31T23:02:10.001Z", 1, 'D', "\"2021-11-02T00:00:00+05:30\""},
		// cross year
		{"America/Los_Angeles", "2022-01-01T02:02:00Z", 1, 'D', "\"2022-01-01T00:00:00-08:00\""},
		{"America/Los_Angeles", "2022-01-01T02:02:10.001Z", 1, 'D', "\"2022-01-01T00:00:00-08:00\""},
		{"Asia/Kolkata", "2021-12-31T23:02:00Z", 1, 'D', "\"2022-01-02T00:00:00+05:30\""},
		{"Asia/Kolkata", "2021-12-31T23:02:10.001Z", 1, 'D', "\"2022-01-02T00:00:00+05:30\""},
	}
	for _, tt := range tests {
		end, _ := time.Parse(time.RFC3339Nano, tt.endDate)
		location, _ := time.LoadLocation(tt.timeZone)
		end = end.UTC().In(location)
		testName := "Test align end date:" + tt.endDate + ", time zone:" + tt.timeZone + ", interval: 1" + string(tt.unit)
		validator := &RequestValidator{
			endDate:  end,
			interval: Interval{tt.interval, tt.unit},
			location: location,
		}
		t.Run(testName, func(t *testing.T) {
			validator.alignEndDate()
			tmpDate, _ := json.Marshal(validator.endDate)
			actual := string(tmpDate)

			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("actual %v, expect %v", actual, tt.expected)
			}
		})
	}
}

func TestToInterval(t *testing.T) {
	tests := []struct {
		in       string
		expected *Interval
	}{
		{"1m", &Interval{1, QueryIntervalUnitMinute}},
		{"20h", &Interval{20, QueryIntervalUnitHour}},
		{"28D", &Interval{28, QueryIntervalUnitDay}},
		{"51W", &Interval{51, QueryIntervalUnitWeek}},
		{"10M", &Interval{10, QueryIntervalUnitMonth}},

		{"100M", nil},
		{"2Y", nil},
		{"", nil},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Parse %s", tt.in)
		t.Run(testName, func(t *testing.T) {
			actual := ToInterval(tt.in)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestAlignWeek(t *testing.T) {
	var tests = []struct {
		weekStartFromSunday bool
		timeZone            string
		startDate           string
		endDate             string
		interval            int
		unit                QueryIntervalUnit
		expectedStart       string
		expectedEnd         string
	}{
		// 1) normal
		{true, "Asia/Chongqing", "2021-11-06T16:00:00Z", "2021-11-13T16:00:00Z", 1, 'W', "\"2021-11-07T00:00:00+08:00\"", "\"2021-11-14T00:00:00+08:00\""},
		{true, "America/Los_Angeles", "2021-11-07T07:00:00Z", "2021-11-14T07:00:00Z", 1, 'W', "\"2021-11-07T00:00:00-07:00\"", "\"2021-11-14T00:00:00-08:00\""},
		{true, "Asia/Kolkata", "2021-11-06T18:30:00Z", "2021-11-13T16:30:00Z", 1, 'W', "\"2021-11-07T00:00:00+05:30\"", "\"2021-11-14T00:00:00+05:30\""},

		{false, "Asia/Chongqing", "2021-11-07T16:00:00Z", "2021-11-14T16:00:00Z", 1, 'W', "\"2021-11-08T00:00:00+08:00\"", "\"2021-11-15T00:00:00+08:00\""},
		{false, "America/Los_Angeles", "2021-11-08T08:00:00Z", "2021-11-15T08:00:00Z", 1, 'W', "\"2021-11-08T00:00:00-08:00\"", "\"2021-11-15T00:00:00-08:00\""},
		{false, "Asia/Kolkata", "2021-11-07T18:30:00Z", "2021-11-14T18:30:00Z", 1, 'W', "\"2021-11-08T00:00:00+05:30\"", "\"2021-11-15T00:00:00+05:30\""},

		// 2) cross week
		{true, "Asia/Chongqing", "2021-11-08T16:00:00Z", "2021-11-16T16:00:00Z", 1, 'W', "\"2021-11-07T00:00:00+08:00\"", "\"2021-11-21T00:00:00+08:00\""},
		{true, "America/Los_Angeles", "2021-11-09T07:00:00Z", "2021-11-16T07:00:00Z", 1, 'W', "\"2021-11-07T00:00:00-07:00\"", "\"2021-11-21T00:00:00-08:00\""},
		{true, "Asia/Kolkata", "2021-11-08T18:30:00Z", "2021-11-16T16:30:00Z", 1, 'W', "\"2021-11-07T00:00:00+05:30\"", "\"2021-11-21T00:00:00+05:30\""},

		{false, "Asia/Chongqing", "2021-11-09T16:00:00Z", "2021-11-18T16:00:00Z", 1, 'W', "\"2021-11-08T00:00:00+08:00\"", "\"2021-11-22T00:00:00+08:00\""},
		{false, "America/Los_Angeles", "2021-11-09T08:00:00Z", "2021-11-18T08:00:00Z", 1, 'W', "\"2021-11-08T00:00:00-08:00\"", "\"2021-11-22T00:00:00-08:00\""},
		{false, "Asia/Kolkata", "2021-11-09T18:30:00Z", "2021-11-18T18:30:00Z", 1, 'W', "\"2021-11-08T00:00:00+05:30\"", "\"2021-11-22T00:00:00+05:30\""},

		{true, "Asia/Chongqing", "2021-11-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'W', "\"2021-10-31T00:00:00+08:00\"", "\"2021-11-14T00:00:00+08:00\""},
		{true, "America/Los_Angeles", "2021-11-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'W', "\"2021-10-31T00:00:00-07:00\"", "\"2021-11-14T00:00:00-08:00\""},
		{true, "Asia/Kolkata", "2021-11-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'W', "\"2021-10-31T00:00:00+05:30\"", "\"2021-11-14T00:00:00+05:30\""},

		{false, "Asia/Chongqing", "2021-11-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'W', "\"2021-11-01T00:00:00+08:00\"", "\"2021-11-15T00:00:00+08:00\""},
		{false, "America/Los_Angeles", "2021-11-02T12:02:10Z", "2021-11-09T12:02:10Z", 1, 'W', "\"2021-11-01T00:00:00-07:00\"", "\"2021-11-15T00:00:00-08:00\""},
		{false, "Asia/Kolkata", "2021-11-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'W', "\"2021-11-01T00:00:00+05:30\"", "\"2021-11-15T00:00:00+05:30\""},

		// 3) cross month
		{true, "Asia/Chongqing", "2021-11-01T18:02:10Z", "2021-11-29T00:02:10Z", 1, 'W', "\"2021-10-31T00:00:00+08:00\"", "\"2021-12-05T00:00:00+08:00\""},
		{true, "America/Los_Angeles", "2021-11-01T12:02:10Z", "2021-11-29T12:02:10Z", 1, 'W', "\"2021-10-31T00:00:00-07:00\"", "\"2021-12-05T00:00:00-08:00\""},
		{true, "Asia/Kolkata", "2021-11-01T00:02:10Z", "2021-11-29T00:02:10Z", 1, 'W', "\"2021-10-31T00:00:00+05:30\"", "\"2021-12-05T00:00:00+05:30\""},

		{false, "Asia/Chongqing", "2021-10-02T00:02:10Z", "2021-11-30T00:02:10Z", 1, 'W', "\"2021-09-27T00:00:00+08:00\"", "\"2021-12-06T00:00:00+08:00\""},
		{false, "America/Los_Angeles", "2021-10-02T12:02:10Z", "2021-11-30T12:02:10Z", 1, 'W', "\"2021-09-27T00:00:00-07:00\"", "\"2021-12-06T00:00:00-08:00\""},
		{false, "Asia/Kolkata", "2021-10-02T00:02:10Z", "2021-11-30T00:02:10Z", 1, 'W', "\"2021-09-27T00:00:00+05:30\"", "\"2021-12-06T00:00:00+05:30\""},

		// 4) cross year
		{true, "Asia/Chongqing", "2021-01-02T00:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-03T00:00:00+08:00\"", "\"2022-01-02T00:00:00+08:00\""},
		{true, "America/Los_Angeles", "2021-01-02T12:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-03T00:00:00-08:00\"", "\"2022-01-02T00:00:00-08:00\""},
		{true, "Asia/Kolkata", "2021-01-02T00:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-03T00:00:00+05:30\"", "\"2022-01-02T00:00:00+05:30\""},

		{true, "Asia/Chongqing", "2021-01-05T00:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-03T00:00:00+08:00\"", "\"2022-01-02T00:00:00+08:00\""},
		{true, "America/Los_Angeles", "2021-01-05T12:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-03T00:00:00-08:00\"", "\"2022-01-02T00:00:00-08:00\""},
		{true, "Asia/Kolkata", "2021-01-05T00:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-03T00:00:00+05:30\"", "\"2022-01-02T00:00:00+05:30\""},

		{false, "Asia/Chongqing", "2021-01-01T00:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-04T00:00:00+08:00\"", "\"2022-01-03T00:00:00+08:00\""},
		{false, "America/Los_Angeles", "2021-01-01T12:02:10Z", "2021-12-28T12:02:10Z", 1, 'W', "\"2021-01-04T00:00:00-08:00\"", "\"2022-01-03T00:00:00-08:00\""},
		{false, "Asia/Kolkata", "2021-01-01T00:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-01-04T00:00:00+05:30\"", "\"2022-01-03T00:00:00+05:30\""},
		{false, "Asia/Chongqing", "2021-07-03T02:02:10Z", "2021-12-28T00:02:10Z", 1, 'W', "\"2021-06-28T00:00:00+08:00\"", "\"2022-01-03T00:00:00+08:00\""},
	}
	for _, tt := range tests {
		location, _ := time.LoadLocation(tt.timeZone)
		start, _ := time.Parse(time.RFC3339Nano, tt.startDate)
		start = start.UTC().In(location)
		end, _ := time.Parse(time.RFC3339Nano, tt.endDate)
		end = end.UTC().In(location)
		testName := "Test  time zone:" + tt.timeZone + ", align start date:" + tt.startDate + ", align end date:" + tt.endDate
		validator := &RequestValidator{
			request:   &Request{weekStartFromSunday: tt.weekStartFromSunday},
			startDate: start,
			endDate:   end,
			interval:  Interval{tt.interval, tt.unit},
			location:  location,
		}
		if tt.weekStartFromSunday {
			validator.weekStart = time.Sunday
		} else {
			validator.weekStart = time.Monday
		}

		t.Run(testName, func(t *testing.T) {
			validator.alignStartDate()
			tmpDate, _ := json.Marshal(validator.startDate)
			actual := string(tmpDate)
			if !reflect.DeepEqual(actual, tt.expectedStart) {
				t.Errorf("actual %v, expect %v", actual, tt.expectedStart)
			}

			validator.alignEndDate()
			tmpDate, _ = json.Marshal(validator.endDate)
			actual = string(tmpDate)
			if !reflect.DeepEqual(actual, tt.expectedEnd) {
				t.Errorf("actual %v, expect %v", actual, tt.expectedEnd)
			}
		})
	}
}

func TestAlignMonth(t *testing.T) {
	var tests = []struct {
		timeZone      string
		startDate     string
		endDate       string
		interval      int
		unit          QueryIntervalUnit
		expectedStart string
		expectedEnd   string
	}{
		// 1) normal
		{"Asia/Chongqing", "2021-09-30T16:00:00Z", "2021-10-31T16:00:00Z", 1, 'M', "\"2021-10-01T00:00:00+08:00\"", "\"2021-11-01T00:00:00+08:00\""},
		{"America/Los_Angeles", "2021-10-01T07:00:00Z", "2021-11-01T07:00:00Z", 1, 'M', "\"2021-10-01T00:00:00-07:00\"", "\"2021-11-01T00:00:00-07:00\""},
		{"Asia/Kolkata", "2021-09-30T18:30:00Z", "2021-10-31T18:30:00Z", 1, 'M', "\"2021-10-01T00:00:00+05:30\"", "\"2021-11-01T00:00:00+05:30\""},

		// 2) cross month
		{"Asia/Chongqing", "2021-10-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'M', "\"2021-10-01T00:00:00+08:00\"", "\"2021-12-01T00:00:00+08:00\""},
		{"America/Los_Angeles", "2021-10-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'M', "\"2021-10-01T00:00:00-07:00\"", "\"2021-12-01T00:00:00-08:00\""},
		{"Asia/Kolkata", "2021-10-02T00:02:10Z", "2021-11-09T00:02:10Z", 1, 'M', "\"2021-10-01T00:00:00+05:30\"", "\"2021-12-01T00:00:00+05:30\""},
		{"Asia/Chongqing", "2021-11-06T16:00:00Z", "2021-12-06T16:00:00Z", 1, 'M', "\"2021-11-01T00:00:00+08:00\"", "\"2022-01-01T00:00:00+08:00\""},

		{"Asia/Chongqing", "2021-10-02T16:00:00Z", "2021-10-31T16:00:00Z", 1, 'M', "\"2021-10-01T00:00:00+08:00\"", "\"2021-11-01T00:00:00+08:00\""},
		{"America/Los_Angeles", "2021-10-02T07:00:00Z", "2021-11-01T07:00:00Z", 1, 'M', "\"2021-10-01T00:00:00-07:00\"", "\"2021-11-01T00:00:00-07:00\""},
		{"Asia/Kolkata", "2021-10-02T18:30:00Z", "2021-10-31T18:30:00Z", 1, 'M', "\"2021-10-01T00:00:00+05:30\"", "\"2021-11-01T00:00:00+05:30\""},

		// 2) cross year
		{"Asia/Chongqing", "2021-10-02T00:02:10Z", "2021-12-09T00:02:10Z", 1, 'M', "\"2021-10-01T00:00:00+08:00\"", "\"2022-01-01T00:00:00+08:00\""},
		{"America/Los_Angeles", "2021-10-02T00:02:10Z", "2021-12-09T00:02:10Z", 1, 'M', "\"2021-10-01T00:00:00-07:00\"", "\"2022-01-01T00:00:00-08:00\""},
		{"Asia/Kolkata", "2021-10-02T00:02:10Z", "2021-12-09T00:02:10Z", 1, 'M', "\"2021-10-01T00:00:00+05:30\"", "\"2022-01-01T00:00:00+05:30\""},
	}
	for _, tt := range tests {
		location, _ := time.LoadLocation(tt.timeZone)
		start, _ := time.Parse(time.RFC3339Nano, tt.startDate)
		start = start.UTC().In(location)
		end, _ := time.Parse(time.RFC3339Nano, tt.endDate)
		end = end.UTC().In(location)
		testName := "Test  time zone:" + tt.timeZone + ", align start date:" + tt.startDate + ", align end date:" + tt.endDate
		validator := &RequestValidator{
			startDate: start,
			endDate:   end,
			interval:  Interval{tt.interval, tt.unit},
			location:  location,
		}

		t.Run(testName, func(t *testing.T) {
			validator.alignStartDate()
			tmpDate, _ := json.Marshal(validator.startDate)
			actual := string(tmpDate)
			if !reflect.DeepEqual(actual, tt.expectedStart) {
				t.Errorf("actual %v, expect %v", actual, tt.expectedStart)
			}

			validator.alignEndDate()
			tmpDate, _ = json.Marshal(validator.endDate)
			actual = string(tmpDate)
			if !reflect.DeepEqual(actual, tt.expectedEnd) {
				t.Errorf("actual %v, expect %v", actual, tt.expectedEnd)
			}
		})
	}
}
