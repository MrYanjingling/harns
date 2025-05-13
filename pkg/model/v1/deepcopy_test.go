package v1

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCopyThingType_WithPointerSlice(t *testing.T) {
	var tests = []struct {
		in ThingType
	}{
		{
			ThingType{
				Name: "admin",
				Characteristics: []*Characteristic{{
					Name:         "length",
					Length:       255,
					DataType:     DataTypeInt,
					DefaultValue: "20",
					Searchable:   false,
				}},
			},
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Copy thing type")
		t.Run(testName, func(t *testing.T) {
			actual := *tt.in.DeepCopyObject().(*ThingType)

			t.Logf("%p, %p", &tt.in, &actual)
			assert.Equal(t, tt.in, actual)
			assert.NotSame(t, &tt.in, &actual)

			t.Logf("%p, %p", tt.in.Characteristics, actual.Characteristics)
			assert.Equal(t, tt.in.Characteristics, actual.Characteristics)
			assert.NotSame(t, tt.in.Characteristics, actual.Characteristics)

			t.Logf("%p, %p", &tt.in.Characteristics[0], &actual.Characteristics[0])
			assert.Equal(t, tt.in.Characteristics[0], actual.Characteristics[0])
			assert.NotSame(t, &tt.in.Characteristics[0], &actual.Characteristics[0])

			t.Logf("%p, %p", &tt.in.Characteristics[0].Name, &actual.Characteristics[0].Name)
			assert.Equal(t, tt.in.Characteristics[0].Name, actual.Characteristics[0].Name)
			assert.NotSame(t, &tt.in.Characteristics[0].Name, &actual.Characteristics[0].Name)
		})
	}
}

func TestCopyThing_WithValueSlice(t *testing.T) {
	var tests = []struct {
		in Thing
	}{
		{
			Thing{
				Name:            "admin",
				Characteristics: []Value{{"length", "20"}},
			},
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Copy thing")
		t.Run(testName, func(t *testing.T) {
			actual := *tt.in.DeepCopyObject().(*Thing)

			t.Logf("%p, %p", &tt.in, &actual)
			assert.Equal(t, tt.in, actual)
			assert.NotSame(t, &tt.in, &actual)

			t.Logf("%p, %p", tt.in.Characteristics, actual.Characteristics)
			assert.Equal(t, tt.in.Characteristics, actual.Characteristics)
			assert.NotSame(t, tt.in.Characteristics, actual.Characteristics)

			t.Logf("%p, %p", &tt.in.Characteristics[0], &actual.Characteristics[0])
			assert.Equal(t, tt.in.Characteristics[0], actual.Characteristics[0])
			assert.NotSame(t, &tt.in.Characteristics[0], &actual.Characteristics[0])

			t.Logf("%p, %p", &tt.in.Characteristics[0].Name, &actual.Characteristics[0].Name)
			assert.Equal(t, tt.in.Characteristics[0].Name, actual.Characteristics[0].Name)
			assert.NotSame(t, &tt.in.Characteristics[0].Name, &actual.Characteristics[0].Name)
		})
	}
}
