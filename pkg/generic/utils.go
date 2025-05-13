package generic

import (
	"fmt"
	"math"
	"net/http"
	"reflect"
)

const (
	// https://docs.influxdata.com/influxdb/v1.8/troubleshooting/frequently-asked-questions/#what-are-the-minimum-and-maximum-timestamps-that-influxdb-can-store
	MaxTTLDay     = 106751
	MaxTTLSeconds = 9223372036
)

func RemoveStringWithOrder(s []string, i int) []string {
	return append(s[:i], s[i+1:]...)
}

func RemoveString(s []string, i int) []string {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func GetRPName(ttl int) string {
	if ttl == math.MaxInt {
		return ""
	} else {
		return fmt.Sprintf("-%d", ttl)
	}
}

func GetTTLDay(ttl int) int {
	if ttl == math.MaxInt {
		return MaxTTLDay
	}
	return ttl
}

func GetScheme(req *http.Request) string {
	if req != nil && req.TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

// DifferenceAndIntersectionStrings  O(len(src) + len(des))
func DifferenceAndIntersectionStrings(src, des []string) (onlySrc, intersection, onlyDes []string) {
	m := make(map[string]uint8)
	for _, k := range src {
		m[k] |= 1 << 0
	}
	for _, k := range des {
		m[k] |= 1 << 1
	}

	for k, v := range m {
		a := v&(1<<0) != 0
		b := v&(1<<1) != 0
		switch {
		case a && b:
			intersection = append(intersection, k)
		case a && !b:
			onlySrc = append(onlySrc, k)
		case !a && b:
			onlyDes = append(onlyDes, k)
		}
	}

	return
}

type getKeyFunc func(value interface{}) string

// DifferenceAndIntersectionSameTypeObjects  O(len(src) + len(des))
func DifferenceAndIntersectionSameTypeObjects(src, des interface{}, get getKeyFunc) (onlySrc, intersection, onlyDes []string) {
	s := reflect.ValueOf(src)
	d := reflect.ValueOf(des)
	m := make(map[string]uint8)

	for i := 0; i < s.Len(); i++ {
		m[get(s.Index(i).Interface())] |= 1 << 0
	}
	for i := 0; i < d.Len(); i++ {
		m[get(d.Index(i).Interface())] |= 1 << 1
	}

	for k, v := range m {
		a := v&(1<<0) != 0
		b := v&(1<<1) != 0
		switch {
		case a && b:
			intersection = append(intersection, k)
		case a && !b:
			onlySrc = append(onlySrc, k)
		case !a && b:
			onlyDes = append(onlyDes, k)
		}
	}

	return
}

func DifferenceAndIntersectionObjects(src, des interface{}, getSrcKey, getDesKey getKeyFunc) (onlySrc, intersection, onlyDes []string) {
	s := reflect.ValueOf(src)
	d := reflect.ValueOf(des)
	m := make(map[string]uint8)

	for i := 0; i < s.Len(); i++ {
		m[getSrcKey(s.Index(i).Interface())] |= 1 << 0
	}
	for i := 0; i < d.Len(); i++ {
		m[getDesKey(d.Index(i).Interface())] |= 1 << 1
	}

	for k, v := range m {
		a := v&(1<<0) != 0
		b := v&(1<<1) != 0
		switch {
		case a && b:
			intersection = append(intersection, k)
		case a && !b:
			onlySrc = append(onlySrc, k)
		case !a && b:
			onlyDes = append(onlyDes, k)
		}
	}

	return
}
