package repository

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type Query struct {
	Filter     Filter               `json:"filter,omitempty"`
	Skip       *int                 `json:"skip,omitempty"`
	Limit      *int                 `json:"limit,omitempty"`
	Sort       map[string]SortOrder `json:"sort,omitempty"`
	Projection map[string]bool      `json:"projection,omitempty"`
	Distinct   bool                 `json:"distinct,omitempty"`
}

func NewQuery() *Query {
	return &Query{
		Sort:       make(map[string]SortOrder),
		Projection: make(map[string]bool),
	}
}

type Filter interface {
	Test(value Object) bool
	ContainsField(key string) bool
}

type And []Filter
type Or []Filter
type Binary struct {
	Key string
	Op  Operator
}

type Operator interface {
	Test(input any) bool
}

type Eq any
type Ne any
type Gt any
type Gte any
type Lt any
type Lte any
type Start string
type End string
type Contains string
type Regexp string
type Exist bool
type In []any
type Nin []any
