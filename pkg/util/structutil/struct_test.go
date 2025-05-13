package structutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMap(t *testing.T) {
	var T = struct {
		A string
		B int
		C bool
	}{
		A: "a-value",
		B: 2,
		C: true,
	}

	expected := map[string]interface{} {
		"A": "a-value",
		"B": 2,
		"C": true,
	}

	t.Run("Struct to map", func(t *testing.T) {
		actual := Map(T)
		assert.Equal(t, expected, actual)
	})

}

func TestNestedMap(t *testing.T) {
	var T = struct {
		A string
		B int
		C struct{
			D bool
		}
	}{
		A: "a-value",
		B: 2,
		C: struct{ D bool }{D: true},
	}

	expected := map[string]interface{} {
		"A": "a-value",
		"B": 2,
		"C": map[string]interface{}{
			"D": true,
		},
	}

	t.Run("Nested struct to map", func(t *testing.T) {
		actual := Map(T)
		assert.Equal(t, expected, actual)
	})
}