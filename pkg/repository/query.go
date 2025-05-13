package repository

import (
	"reflect"
	"regexp"
	"strings"
)

type Object map[string]any
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type Query struct {
	Filters    Filters              `json:"filters,omitempty"`
	Skip       *int                 `json:"skip,omitempty"`
	Limit      *int                 `json:"limit,omitempty"`
	Sort       map[string]SortOrder `json:"sort,omitempty"`
	Projection map[string]bool      `json:"projection,omitempty"`
	Distinct   bool                 `json:"distinct,omitempty"`
}

func NewQuery() *Query {
	return &Query{
		Filters:    NewFilters(),
		Sort:       make(map[string]SortOrder),
		Projection: make(map[string]bool),
	}
}

type Filters []Filter

func NewFilters() Filters {
	return make(Filters, 0)
}

func (f Filters) MatchValue(value Object) bool {
	for _, filter := range f {
		if !filter.MatchValue(value) {
			return false
		}
	}
	return true
}

func (f Filters) IsEmpty() bool {
	return len(f) == 0
}

func (f Filters) ContainsField(key string) bool {
	for _, filter := range f {
		if filter.ContainsField(key) {
			return true
		}
	}
	return false
}

type Filter interface {
	MatchValue(value Object) bool
	ContainsField(key string) bool
}

type FieldFilter struct {
	Key   string
	Op    Operator
	Value any
}

func (f FieldFilter) MatchValue(value Object) bool {
	val, ok := value[f.Key]
	if !ok {
		return false
	}
	return f.Op.MatchValue(val, f.Value)
}

func (f FieldFilter) ContainsField(key string) bool {
	return f.Key == key
}

type Operator interface {
	MatchValue(val any, expected any) bool
}

type EQ struct{}

func (op EQ) MatchValue(val any, expected any) bool {
	return val == expected
}

type NE struct{}

func (op NE) MatchValue(val any, expected any) bool {
	return val != expected
}

type GT struct{}

func (op GT) MatchValue(val any, expected any) bool {
	switch v := val.(type) {
	case float64:
		switch e := expected.(type) {
		case float64:
			return v > e
		case int:
			return v > float64(e)
		case float32:
			return v > float64(e)
		}
	case int:
		switch e := expected.(type) {
		case float64:
			return float64(v) > e
		case int:
			return v > e
		case float32:
			return float64(v) > float64(e)
		}
	case float32:
		switch e := expected.(type) {
		case float64:
			return float64(v) > e
		case int:
			return float64(v) > float64(e)
		case float32:
			return v > e
		}
	}
	return false
}

type GTE struct{}

func (op GTE) MatchValue(val any, expected any) bool {
	switch v := val.(type) {
	case float64:
		switch e := expected.(type) {
		case float64:
			return v >= e
		case int:
			return v >= float64(e)
		case float32:
			return v >= float64(e)
		}
	case int:
		switch e := expected.(type) {
		case float64:
			return float64(v) >= e
		case int:
			return v >= e
		case float32:
			return float64(v) >= float64(e)
		}
	case float32:
		switch e := expected.(type) {
		case float64:
			return float64(v) >= e
		case int:
			return float64(v) >= float64(e)
		case float32:
			return v >= e
		}
	}
	return false
}

type LT struct{}

func (op LT) MatchValue(val any, expected any) bool {
	switch v := val.(type) {
	case float64:
		switch e := expected.(type) {
		case float64:
			return v < e
		case int:
			return v < float64(e)
		case float32:
			return v < float64(e)
		}
	case int:
		switch e := expected.(type) {
		case float64:
			return float64(v) < e
		case int:
			return v < e
		case float32:
			return float64(v) < float64(e)
		}
	case float32:
		switch e := expected.(type) {
		case float64:
			return float64(v) < e
		case int:
			return float64(v) < float64(e)
		case float32:
			return v < e
		}
	}
	return false
}

type LTE struct{}

func (op LTE) MatchValue(val any, expected any) bool {
	switch v := val.(type) {
	case float64:
		switch e := expected.(type) {
		case float64:
			return v <= e
		case int:
			return v <= float64(e)
		case float32:
			return v <= float64(e)
		}
	case int:
		switch e := expected.(type) {
		case float64:
			return float64(v) <= e
		case int:
			return v <= e
		case float32:
			return float64(v) <= float64(e)
		}
	case float32:
		switch e := expected.(type) {
		case float64:
			return float64(v) <= e
		case int:
			return float64(v) <= float64(e)
		case float32:
			return v <= e
		}
	}
	return false
}

type START struct{}

func (op START) MatchValue(val any, expected any) bool {
	s, ok := val.(string)
	prefix, ok := expected.(string)
	return ok && strings.HasPrefix(s, prefix)
}

type END struct{}

func (op END) MatchValue(val any, expected any) bool {
	s, ok := val.(string)
	suffix, ok := expected.(string)
	return ok && strings.HasSuffix(s, suffix)
}

type CONTAINS struct{}

func (op CONTAINS) MatchValue(val any, expected any) bool {
	// 支持字符串和字符串数组的CONTAINS判断
	switch v := val.(type) {
	case string:
		substr, ok := expected.(string)
		return ok && strings.Contains(v, substr)
	default:
		// 使用反射处理数组/切片类型的CONTAINS判断
		valVal := reflect.ValueOf(val)
		if valVal.Kind() == reflect.Slice || valVal.Kind() == reflect.Array {
			for i := 0; i < valVal.Len(); i++ {
				if reflect.DeepEqual(valVal.Index(i).Interface(), expected) {
					return true
				}
			}
		}
	}
	return false
}

type REGEXP struct{}

func (op REGEXP) MatchValue(val any, expected any) bool {
	s, ok := val.(string)
	pattern, ok := expected.(string)
	if !ok {
		return false
	}
	r, err := regexp.Compile(pattern)
	return err == nil && r.MatchString(s)
}

type IN struct{}

func (op IN) MatchValue(val any, expected any) bool {
	values, ok := expected.([]any)
	if !ok {
		return false
	}
	for _, v := range values {
		if v == val {
			return true
		}
	}
	return false
}

type NIN struct{}

func (op NIN) MatchValue(val any, expected any) bool {
	values, ok := expected.([]any)
	if !ok {
		return false
	}
	for _, v := range values {
		if v == val {
			return false
		}
	}
	return true
}

type AndFilter struct {
	Filters Filters
}

func (f AndFilter) MatchValue(value Object) bool {
	return f.Filters.MatchValue(value)
}

func (f AndFilter) ContainsField(key string) bool {
	return f.Filters.ContainsField(key)
}

type OrFilter struct {
	Filters Filters
}

func (f OrFilter) MatchValue(value Object) bool {
	for _, filter := range f.Filters {
		if filter.MatchValue(value) {
			return true
		}
	}
	return false
}

func (f OrFilter) ContainsField(key string) bool {
	return f.Filters.ContainsField(key)
}
