package aesutil

import (
	"fmt"
	"reflect"
	"testing"
)

func TestCBC(t *testing.T) {

	var tests = []struct{
		in     string
	}{
		{"abcdefghijklmnopqrstuvwxyz123456"},
		{"abcdefghijklmnopqrstuvwxyz12345678"},
		{"abcdefghijklmnopqrstuvwxyz"},
		{"a"},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Encrypt/Decript content %s", tt.in)
		t.Run(testName, func(t *testing.T) {
			key := "test"
			ciphertext := EncryptCBC(tt.in, key)
			actual := DecryptCBC(ciphertext, key)
			if !reflect.DeepEqual(string(actual), tt.in) {
				t.Errorf("actual %v, expect %v", string(actual), tt.in)
			}
		})
	}
}

func TestFormatKey(t *testing.T) {
	var tests = []struct{
		in     string
		expect []byte
	}{
		{"a", []byte("a\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F\x1F")},
		{"abcdefghijklmnopqrstuvwxyz123456", []byte("abcdefghijklmnopqrstuvwxyz123456")},
		{"abcdefghijklmnopqrstuvwxyz12345678", []byte("abcdefghijklmnopqrstuvwxyz123456")},
		{"abcdefghijklmnopqrstuvwxyz", []byte("abcdefghijklmnopqrstuvwxyz\x06\x06\x06\x06\x06\x06")},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Format key %s", tt.in)
		t.Run(testName, func(t *testing.T) {
			actual := formatKey([]byte(tt.in))
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}
