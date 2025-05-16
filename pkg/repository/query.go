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
	Filter     Filter               `json:"filter,omitempty"`
	Skip       uint64               `json:"skip,omitempty"`
	Limit      uint64               `json:"limit,omitempty"`
	Sort       map[string]SortOrder `json:"sort,omitempty"`
	Projection map[string]struct{}  `json:"projection,omitempty"`
	Distinct   bool                 `json:"distinct,omitempty"`
}

func NewQuery() *Query {
	return &Query{
		Sort:       make(map[string]SortOrder),
		Projection: make(map[string]struct{}),
	}
}

type Filter interface {
	Test(value Object) bool
	ContainsField(key string) bool
}

type And []Filter

func (a And) Test(value Object) bool {
	for _, filter := range a {
		if !filter.Test(value) {
			return false
		}
	}
	return true
}

func (a And) ContainsField(key string) bool {
	for _, filter := range a {
		if filter.ContainsField(key) {
			return true
		}
	}
	return false
}

type Or []Filter

func (o Or) Test(value Object) bool {
	for _, filter := range o {
		if filter.Test(value) {
			return true
		}
	}
	return false
}

func (o Or) ContainsField(key string) bool {
	for _, filter := range o {
		if filter.ContainsField(key) {
			return true
		}
	}
	return false
}

type Binary struct {
	Key string
	Op  Operator
}

func (b *Binary) Test(value Object) bool {
	return b.Op.Test(b.Key, value)
}

func (b *Binary) ContainsField(key string) bool {
	return b.Key == key
}

type Operator interface {
	Test(key string, value Object) bool
}

type Eq struct {
	Value any
}

func (e *Eq) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	return val == e.Value
}

type Ne struct {
	Value any
}

func (n *Ne) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return true
	}
	return !eq(val, n.Value)
}

type Gt struct {
	Value any
}

func (g *Gt) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	return gt(val, g.Value)
}

type Gte struct {
	Value any
}

func (g *Gte) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	return gte(val, g.Value)
}

type Lt struct {
	Value any
}

func (l *Lt) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	return lt(val, l.Value)
}

type Lte struct {
	Value any
}

func (l *Lte) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	return lte(val, l.Value)
}

type Start string

func (s *Start) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(str, string(*s))
}

type End string

func (e *End) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	return strings.HasSuffix(str, string(*e))
}

type Contains string

func (c *Contains) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	return strings.Contains(str, string(*c))
}

type Regexp string

func (r *Regexp) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	re, err := regexp.Compile(string(*r))
	if err != nil {
		return false
	}
	return re.MatchString(str)
}

type Exist bool

func (e *Exist) Test(key string, value Object) bool {
	_, ok := value[key]
	return ok == bool(*e)
}

type In []any

func (i *In) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return false
	}
	for _, v := range *i {
		if eq(v, val) {
			return true
		}
	}
	return false
}

type Nin []any

func (n *Nin) Test(key string, value Object) bool {
	val, ok := value[key]
	if !ok {
		return true
	}
	for _, v := range *n {
		if eq(v, val) {
			return false
		}
	}
	return true
}
