package event

import (
	"fmt"
	"lightiot/pkg/event/runtime"
	v1 "lightiot/pkg/event/v1"
	"reflect"
	"testing"
)

func TestParseFieldValues(t *testing.T) {
	var tests = []struct {
		in     *v1.Field
		expect *runtime.Field
	}{
		{&v1.Field{DataType: v1.DatatypeEnum}, nil},
		{&v1.Field{DataType: v1.DatatypeEnum, Values: ""}, nil},
		{&v1.Field{DataType: v1.DatatypeEnum, Values: []string{"Up", "Idle", "关机"}}, &runtime.Field{DataType: v1.DatatypeEnum, Values: []interface{}{"Up", "Idle", "关机"}}},
		{&v1.Field{DataType: v1.DatatypeMap}, nil},
		{&v1.Field{DataType: v1.DatatypeMap, Values: map[string]string{"1": "Up", "2": "Idle", "3": "关机"}}, &runtime.Field{DataType: v1.DatatypeMap, Values: map[string]interface{}{"1": "Up", "2": "Idle", "3": "关机"}}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Parse %s values", tt.in.DataType)
		t.Run(testName, func(t *testing.T) {
			f := tt.in
			actual, _ := parseFieldValues(f)
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}
