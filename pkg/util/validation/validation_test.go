package validation

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"unicode/utf8"
)

func TestIsStringLenInRange(t *testing.T) {
	var tests = []struct {
		in     string
		expect bool
	}{
		{"abc", true},
		{"产线", true},
		{"a产b线", true},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%s string length is %d", tt.in, len(tt.in))
		t.Run(testName, func(t *testing.T) {
			actual := IsStringLengthInRange(tt.in, 5, 9)
			assert.Equal(t, tt.expect, actual)
		})
	}
}

func TestIsRuneLenInRange(t *testing.T) {
	var tests = []struct {
		in     string
		expect bool
	}{
		{"abc", true},
		{"产线", true},
		{"a产b线", true},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%s rune length is %d-%d", tt.in, utf8.RuneCountInString(tt.in), len([]rune(tt.in)))
		t.Run(testName, func(t *testing.T) {
			actual := IsRuneLengthInRange(tt.in, 2, 3)
			assert.Equal(t, tt.expect, actual)
		})
	}
}
