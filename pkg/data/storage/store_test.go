package storage

import (
	"fmt"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"reflect"
	"testing"
)

func TestConvertDataAccordingToType(t *testing.T) {
	var tests = []struct {
		p      runtime.Property
		in     interface{}
		expect interface{}
	}{
		{runtime.Property{DataType: v1.DataTypeInt}, "123", 123},
		// can NOT convert float to int
		// {runtime.Property{Datatype: v1.DataTypeInt}, "123.1",123},
		// in 64-bit system, int represents 64 bit;
		{runtime.Property{DataType: v1.DataTypeLong}, "68719476736", int64(68719476736)},
		{runtime.Property{DataType: v1.DataTypeDouble}, "100", float64(100)},
		{runtime.Property{DataType: v1.DataTypeDouble}, "100.123", 100.123},
		{runtime.Property{DataType: v1.DataTypeBoolean}, "true", true},
		{runtime.Property{DataType: v1.DataTypeBoolean}, "1", true},
		{runtime.Property{DataType: v1.DataTypeString, Length: 6}, "abc", "abc"},
		{runtime.Property{DataType: v1.DataTypeString, Length: 6}, "abcedfgh", "abcedf"},
		{runtime.Property{DataType: v1.DataTypeString, Length: 5}, "baby我爱你", "baby我"},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Convert %#v to %s", tt.in, tt.p.DataType)
		t.Run(testName, func(t *testing.T) {
			p := &tt.p
			actual := convertDataAccordingToType(p, tt.in)
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}
