package validation

import (
	"strings"
	"unicode/utf8"
)

func IsStringLengthInRange(value string, min, max int) bool {
	return len(value) >= min && len(value) <= max
}

func IsRuneLengthInRange(value string, min, max int) bool {
	// it is faster than return the length of []rune(value)
	l := utf8.RuneCountInString(value)
	return l >= min && l <= max
}

// IsColumnNameLengthInRange compliant with clickhouse file name
// https://github.com/ClickHouse/ClickHouse/blob/v22.1.2.2-stable/src/Common/escapeForFileName.cpp#L8
func IsColumnNameLengthInRange(value string, min, max int) bool {
	var l int
	chars := []byte(value)
	for _, c := range chars {
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			(c == '_') {
			l += 1
		} else {
			l += 3
		}
	}
	return l >= min && l <= max
}

func IsAlphanumeric(value string) bool {
	return alphaNumericRegex.MatchString(value)
}

func IsEmail(value string) bool {
	return emailRegex.MatchString(value)
}

func ExcludesAll(value string, chars string) bool {
	return !strings.ContainsAny(value, chars)
}

func IsHostnameRFC952(value string) bool {
	return hostnameRegexRFC952.MatchString(value)
}

func IsNumber(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	case string:
		return numberRegex.MatchString(value.(string))
	default:
		return false
	}
}
