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
	Test(record Record) bool
	ContainsField(key string) bool
}

var (
	_ Filter = (And)(nil)
	_ Filter = (Or)(nil)
	_ Filter = (*Binary)(nil)
)

type And []Filter

func (a And) Test(record Record) bool {
	for _, filter := range a {
		if !filter.Test(record) {
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

func (o Or) Test(record Record) bool {
	for _, filter := range o {
		if filter.Test(record) {
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

func (b *Binary) Test(record Record) bool {
	val, _ := record.Get(b.Key)
	return b.Op.Test(val)
}

func (b *Binary) ContainsField(key string) bool {
	return b.Key == key
}

type Operator interface {
	Test(val any) bool
}

var (
	_ Operator = (*Eq)(nil)
	_ Operator = (*Ne)(nil)
	_ Operator = (*Gte)(nil)
	_ Operator = (*Gt)(nil)
	_ Operator = (*Lte)(nil)
	_ Operator = (*Lt)(nil)
	_ Operator = (*In)(nil)
	_ Operator = (*Nin)(nil)
	_ Operator = (*Start)(nil)
	_ Operator = (*End)(nil)
	_ Operator = (*Contains)(nil)
	_ Operator = (*Regexp)(nil)
	_ Operator = (*Exist)(nil)
)

type Eq struct {
	Value any
}

func (e *Eq) Test(val any) bool {
	if val == nil {
		return false
	}
	return val == e.Value
}

type Ne struct {
	Value any
}

func (n *Ne) Test(val any) bool {
	if val == nil {
		return false
	}
	return !eq(val, n.Value)
}

type Gt struct {
	Value any
}

func (g *Gt) Test(val any) bool {
	if val == nil {
		return false
	}
	return gt(val, g.Value)
}

type Gte struct {
	Value any
}

func (g *Gte) Test(val any) bool {
	if val == nil {
		return false
	}
	return gte(val, g.Value)
}

type Lt struct {
	Value any
}

func (l *Lt) Test(val any) bool {
	if val == nil {
		return false
	}
	return lt(val, l.Value)
}

type Lte struct {
	Value any
}

func (l *Lte) Test(val any) bool {
	if val == nil {
		return false
	}
	return lte(val, l.Value)
}

type Start string

func (s *Start) Test(val any) bool {
	if val == nil {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(str, string(*s))
}

type End string

func (e *End) Test(val any) bool {
	if val == nil {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	return strings.HasSuffix(str, string(*e))
}

type Contains string

func (c *Contains) Test(val any) bool {
	if val == nil {
		return false
	}
	str, ok := val.(string)
	if !ok {
		return false
	}
	return strings.Contains(str, string(*c))
}

type Regexp string

func (r *Regexp) Test(val any) bool {
	if val == nil {
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

func (e *Exist) Test(val any) bool {
	return val != nil == bool(*e)
}

type In []any

func (i *In) Test(val any) bool {
	if val == nil {
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

func (n *Nin) Test(val any) bool {
	if val == nil {
		return true
	}
	for _, v := range *n {
		if eq(v, val) {
			return false
		}
	}
	return true
}
