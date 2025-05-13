package bitset

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		in     uint
		expect int
	}{
		{1, 1},
		{64, 1},
		{65, 2},
		{128, 2},
		{129, 3},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("BitSet length %d", tt.in)
		t.Run(testName, func(t *testing.T) {
			l := tt.in
			actual := New(l)
			if len(actual.set) != tt.expect {
				t.Errorf("actual %v, expect %v", len(actual.set), tt.expect)
			}
		})
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		in     uint
		index  uint
		expect []uint64
	}{
		{32, 2, []uint64{4}},
		{32, 32, []uint64{0}},
		{32, 33, []uint64{0}},
		{68, 2, []uint64{4, 0}},
		{68, 66, []uint64{0, 4}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Set %d for length %d BitSet", tt.index, tt.in)
		t.Run(testName, func(t *testing.T) {
			actual := New(tt.in)
			actual.Set(tt.index)
			if !reflect.DeepEqual(actual.set, tt.expect) {
				t.Errorf("actual %v, expect %v", actual.set, tt.expect)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		in     uint
		index  uint
		expect string
	}{
		{3, 1, "010"},
		{32, 31, "10000000000000000000000000000000"},
		{68, 31, "00000000000000000000000000000000000010000000000000000000000000000000"},
		{68, 66, "01000000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Set %d for length %d BitSet", tt.index, tt.in)
		t.Run(testName, func(t *testing.T) {
			bs := New(tt.in)
			bs.Set(tt.index)
			actual := bs.String()
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}

func TestNextClear(t *testing.T) {
	tests := []struct {
		in     uint
		index  uint
		start  uint
		expect uint
	}{
		{3, 1, 0,0},
		{32, 6, 6,7},
		{32, 31, 31,0},
		{68, 66, 66,67},
		{68, 66, 64,64},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Get clear from %d", tt.start)
		t.Run(testName, func(t *testing.T) {
			bs := New(tt.in)
			bs.Set(tt.index)
			actual, _ := bs.NextClear(tt.start)
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}