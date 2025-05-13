package job

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestToTimeBlock(t *testing.T) {
	in := time.Date(2021, time.October, 20, 8, 55, 28, 32, time.UTC)

	locs := []string{
		"UTC",
		"Asia/Shanghai",  // +08:00
		"Asia/Kolkata",   // India, +05:30
		"Asia/Kathmandu", // Nepal, +05:45
	}

	times := map[string][]time.Time{
		"UTC": {
			time.Date(2021, time.October, 20, 8, 55, 28, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 55, 0, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 0, 0, 0, time.UTC),
			time.Date(2021, time.October, 20, 0, 0, 0, 0, time.UTC),
		},
		"Asia/Shanghai": {
			time.Date(2021, time.October, 20, 8, 55, 28, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 55, 0, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 0, 0, 0, time.UTC),
			time.Date(2021, time.October, 19, 16, 0, 0, 0, time.UTC),
		},
		"Asia/Kolkata": {
			time.Date(2021, time.October, 20, 8, 55, 28, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 55, 0, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 30, 0, 0, time.UTC),
			time.Date(2021, time.October, 19, 18, 30, 0, 0, time.UTC),
		},
		"Asia/Kathmandu": {
			time.Date(2021, time.October, 20, 8, 55, 28, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 55, 0, 0, time.UTC),
			time.Date(2021, time.October, 20, 8, 15, 0, 0, time.UTC),
			time.Date(2021, time.October, 19, 18, 15, 0, 0, time.UTC),
		},
	}

	for _, l := range locs {
		loc, _ := time.LoadLocation(l)

		func(loc *time.Location) {
			tests := []struct {
				unit     TimeBlockUnit
				expected string
			}{
				{TimeBlockUnitSecond, "1634720128s"},
				{TimeBlockUnitMinute, "27245335m"},
				{TimeBlockUnitHour, "454088h"},
				{TimeBlockUnitDay, "18920d"},
			}

			for _, tt := range tests {
				testName := fmt.Sprintf("%s To %c", loc, TimeBlockUnitToString[tt.unit])
				t.Run(testName, func(t *testing.T) {
					actual := toTimeBlock(in, tt.unit, loc)
					at := actual.ToTime()
					t.Log(actual.String())
					// t.Logf("utc: %v, loc: %v", at.UTC(), at.In(loc))
					assert.Equal(t, times[loc.String()][tt.unit], at)
				})
			}
		}(loc)
	}
}

func TestToTime(t *testing.T) {
	tests := []struct {
		in       timeBlock
		expected time.Time
	}{
		{timeBlock{1634720128, TimeBlockUnitSecond}, time.Date(2021, time.October, 20, 8, 55, 28, 0, time.UTC)},
		{timeBlock{1634720100, TimeBlockUnitMinute}, time.Date(2021, time.October, 20, 8, 55, 0, 0, time.UTC)},
		{timeBlock{1634716800, TimeBlockUnitHour}, time.Date(2021, time.October, 20, 8, 0, 0, 0, time.UTC)},
		{timeBlock{1634659200, TimeBlockUnitDay}, time.Date(2021, time.October, 19, 16, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("From %s", tt.in.String())
		t.Run(testName, func(t *testing.T) {
			actual := tt.in.ToTime()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetProcTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tests := []struct {
		in       time.Duration
		expected int64
	}{
		{time.Second, time.Date(2021, time.November, 9, 10, 12, 53, 0, loc).Unix()},
		{time.Minute, time.Date(2021, time.November, 9, 10, 13, 0, 0, loc).Unix()},
		{time.Hour, time.Date(2021, time.November, 9, 11, 0, 0, 0, loc).Unix()},
		{24 * time.Hour, time.Date(2021, time.November, 10, 0, 0, 0, 0, loc).Unix()},
	}

	now := time.Date(2021, time.November, 9, 2, 12, 52, 339, time.UTC)
	start := time.Date(2021, time.November, 9, 2, 12, 51, 339, time.UTC)

	for _, tt := range tests {
		testName := fmt.Sprintf("Duration %s", tt.in.String())
		t.Run(testName, func(t *testing.T) {
			actual := getProcTime(now, start, tt.in, loc)
			t.Log(tt.expected)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetProcTimeWithOldData(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tests := []struct {
		in       time.Duration
		expected int64
	}{
		{time.Second, time.Date(2021, time.November, 9, 10, 12, 53, 0, loc).Unix()},
		{time.Minute, time.Date(2021, time.November, 9, 10, 13, 0, 0, loc).Unix()},
		{time.Hour, time.Date(2021, time.November, 9, 11, 0, 0, 0, loc).Unix()},
		{24 * time.Hour, time.Date(2021, time.November, 10, 0, 0, 0, 0, loc).Unix()},
	}

	now := time.Date(2021, time.November, 9, 2, 12, 52, 339, time.UTC)
	start := time.Date(2021, time.November, 8, 0, 12, 51, 339, time.UTC)

	for _, tt := range tests {
		testName := fmt.Sprintf("Duration %s", tt.in.String())
		t.Run(testName, func(t *testing.T) {
			actual := getProcTime(now, start, tt.in, loc)
			t.Log(tt.expected)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetLocalTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tests := []struct {
		in       time.Duration
		expected time.Time
	}{
		{time.Second, time.Date(2021, time.November, 9, 10, 12, 0, 0, loc)},
		{time.Minute, time.Date(2021, time.November, 9, 10, 12, 0, 0, loc)},
		{time.Hour, time.Date(2021, time.November, 9, 10, 0, 0, 0, loc)},
		{24 * time.Hour, time.Date(2021, time.November, 9, 0, 0, 0, 0, loc)},
	}

	now := time.Date(2021, time.November, 9, 2, 12, 52, 339, time.UTC)

	for _, tt := range tests {
		testName := fmt.Sprintf("Duration %s", tt.in.String())
		t.Run(testName, func(t *testing.T) {
			actual := getLocalTime(now, tt.in, loc)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetLocalTimeWithZone(t *testing.T) {
	// +08:00
	locs := []string{
		"Asia/Shanghai",  // +08:00
		"Asia/Kolkata",   // India, +05:30
		"Asia/Kathmandu", // Nepal, +05:45
	}
	for _, l := range locs {
		loc, _ := time.LoadLocation(l)

		func(loc *time.Location) {
			tests := []struct {
				in       time.Duration
				expected time.Time
			}{
				{time.Second, time.Date(2021, time.November, 9, 10, 12, 0, 0, loc)},
				{time.Minute, time.Date(2021, time.November, 9, 10, 12, 0, 0, loc)},
				{time.Hour, time.Date(2021, time.November, 9, 10, 0, 0, 0, loc)},
				{24 * time.Hour, time.Date(2021, time.November, 9, 0, 0, 0, 0, loc)},
			}

			now := time.Date(2021, time.November, 9, 10, 12, 52, 339, loc)

			for _, tt := range tests {
				testName := fmt.Sprintf("Duration %s in %s", tt.in.String(), l)
				t.Run(testName, func(t *testing.T) {
					actual := getLocalTime(now, tt.in, loc)
					assert.Equal(t, tt.expected, actual)
				})
			}
		}(loc)
	}
}

func TestDaylightSavings(t *testing.T) {
	for _, location := range []string{"Europe/Amsterdam", "America/New_York", "Australia/Sydney" /*, "Asia/Ulaanbaatar"*/} {

		testName := fmt.Sprintf("Location %s", location)
		t.Run(testName, func(t *testing.T) {
			loc, err := time.LoadLocation(location)
			if err != nil {
				t.Error(err)
			}

			now := time.Now().In(loc)
			z1, timeOffset := now.Zone()
			zw, winterOffset := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc).Zone()
			zs, summerOffset := time.Date(now.Year(), 7, 1, 0, 0, 0, 0, loc).Zone()

			if winterOffset > summerOffset {
				winterOffset, summerOffset = summerOffset, winterOffset
				zw, zs = zs, zw
			}

			t.Log("now:   ", z1, timeOffset)
			t.Log("winter:", zw, winterOffset)
			t.Log("summer:", zs, summerOffset)

			// if winterOffset != summerOffset { // the location has daylight saving
			// 	if timeOffset != winterOffset {
			// 		t.Log("Daylight Saving")
			// 	}
			// }

			assert.NotEqual(t, winterOffset, summerOffset)
		})
	}
}

func benchmarkTTL(f func(), b *testing.B) {
	for n := 0; n < b.N; n++ {
		f()
	}
}

func BenchmarkSelfTTL(b *testing.B) {
	// procTime := time.Date(2021, time.November, 9, 2, 12, 0, 0, time.UTC).Unix()
	benchmarkTTL(func() {
		// _ = time.Duration(procTime - time.Now().Unix()) * time.Second
		if time.Now().Unix() > 1636626522 {

		}
	}, b)
}

func BenchmarkBuiltInTTL(b *testing.B) {
	// procTime := time.Date(2021, time.November, 9, 2, 12, 0, 0, time.UTC).Unix()
	benchmarkTTL(func() {
		// _ = time.Now().Sub(time.Unix(procTime, 0))
		if time.Now().Before(time.Unix(1636626522, 0)) {

		}
	}, b)
}
