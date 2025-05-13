package repository

import (
	"regexp"
	"strings"
)

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

type Filters []Exp

func NewFilters() Filters {
	return make(Filters, 0)
}

func (f Filters) MatchValue(value map[string]any) bool {
	for _, exp := range f {
		if !exp.MatchValue(value) {
			return false
		}
	}
	return true
}

func (f Filters) IsEmpty() bool {
	return len(f) == 0
}

func (f Filters) ContainsField(key string) bool {
	for _, exp := range f {
		if exp.ContainsField(key) {
			return true
		}
	}
	return false
}

type Exp interface {
	MatchValue(value map[string]any) bool
	ContainsField(key string) bool
}

type FieldExp struct {
	Key string
	Ops []Op
}

func (e FieldExp) MatchValue(value map[string]any) bool {
	val, ok := value[e.Key]
	if !ok {
		return false
	}
	for _, op := range e.Ops {
		if !op.MatchValue(val) {
			return false
		}
	}
	return true
}

func (e FieldExp) ContainsField(key string) bool {
	return e.Key == key
}

type AndExp struct {
	Filters Filters
}

func (e AndExp) MatchValue(value map[string]any) bool {
	return e.Filters.MatchValue(value)
}

func (e AndExp) ContainsField(key string) bool {
	return e.Filters.ContainsField(key)
}

type OrExp struct {
	Filters Filters
}

func (e OrExp) MatchValue(value map[string]any) bool {
	for _, exp := range e.Filters {
		if exp.MatchValue(value) {
			return true
		}
	}
	return false
}

func (e OrExp) ContainsField(key string) bool {
	return e.Filters.ContainsField(key)
}

type Op interface {
	MatchValue(val any) bool
}

type EQ struct {
	Value any
}

func (op EQ) MatchValue(val any) bool {
	return val == op.Value
}

type NE struct {
	Value any
}

func (op NE) MatchValue(val any) bool {
	return val != op.Value
}

type GT struct {
	Value float64
}

func (op GT) MatchValue(val any) bool {
	v, ok := val.(float64)
	return ok && v > op.Value
}

type GTE struct {
	Value float64
}

func (op GTE) MatchValue(val any) bool {
	v, ok := val.(float64)
	return ok && v >= op.Value
}

type LT struct {
	Value float64
}

func (op LT) MatchValue(val any) bool {
	v, ok := val.(float64)
	return ok && v < op.Value
}

type LTE struct {
	Value float64
}

func (op LTE) MatchValue(val any) bool {
	v, ok := val.(float64)
	return ok && v <= op.Value
}

type START struct {
	Prefix string
}

func (op START) MatchValue(val any) bool {
	s, ok := val.(string)
	return ok && strings.HasPrefix(s, op.Prefix)
}

type END struct {
	Suffix string
}

func (op END) MatchValue(val any) bool {
	s, ok := val.(string)
	return ok && strings.HasSuffix(s, op.Suffix)
}

type CONTAINS struct {
	Substr string
}

func (op CONTAINS) MatchValue(val any) bool {
	s, ok := val.(string)
	return ok && strings.Contains(s, op.Substr)
}

type REGEXP struct {
	Pattern string
}

func (op REGEXP) MatchValue(val any) bool {
	s, ok := val.(string)
	if !ok {
		return false
	}
	r, err := regexp.Compile(op.Pattern)
	return err == nil && r.MatchString(s)
}

type IN struct {
	Values []any
}

func (op IN) MatchValue(val any) bool {
	for _, v := range op.Values {
		if v == val {
			return true
		}
	}
	return false
}

type NIN struct {
	Values []any
}

func (op NIN) MatchValue(val any) bool {
	for _, v := range op.Values {
		if v == val {
			return false
		}
	}
	return true
}
