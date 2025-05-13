package clickhouse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type compare byte

const (
	equal compare = iota
	greaterAndEqual
	lessAndEqual

	timeFormat        = "2006-01-02 15:04:05.999999"
	defaultQueryLimit = 2000
)

type Query struct {
	selects []string
	from    struct {
		database string
		table    string
	}
	where []*Where
	desc  bool
	limit int
	alias string
}

type Where struct {
	field string
	value interface{}
	cond  compare
}

func NewQuery() *Query {
	return &Query{
		limit: defaultQueryLimit,
	}
}

func (q *Query) Select(selects []string) *Query {
	q.selects = append(q.selects, selects...)
	return q
}

func (q *Query) From(database, table string) *Query {
	q.from.database = database
	q.from.table = table
	return q
}

func (q *Query) Where(where []*Where) *Query {
	q.where = append(q.where, where...)
	return q
}

func (q *Query) Desc(desc bool) *Query {
	q.desc = desc
	return q
}

func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

func Eq(field string, value interface{}) *Where {
	return &Where{
		field: field,
		value: value,
		cond:  equal,
	}
}

func GE(field string, value interface{}) *Where {
	return &Where{
		field: field,
		value: value,
		cond:  greaterAndEqual,
	}
}

func LE(field string, value interface{}) *Where {
	return &Where{
		field: field,
		value: value,
		cond:  lessAndEqual,
	}
}

func (q *Query) Table() string {
	return q.from.table
}

func (q *Query) Selects() []string {
	return q.selects
}

func (w *Where) string() string {
	var cond, ret string
	switch w.cond {
	case equal:
		cond = "="
	case greaterAndEqual:
		cond = ">="
	case lessAndEqual:
		cond = "<="
	}

	convertBool := func(tf bool) (r int) {
		if tf {
			r = 1
		}
		return
	}

	switch v := w.value.(type) {
	case string:
		ret = fmt.Sprintf(`"%s" %s '%s'`, w.field, cond, v)
	case time.Time:
		ret = fmt.Sprintf(`"%s" %s toDateTime64('%s', 6, 'UTC')`, w.field, cond, v.UTC().Format(timeFormat))
	case bool:
		ret = fmt.Sprintf(`"%s" %s %d`, w.field, cond, convertBool(v))
	default:
		ret = fmt.Sprintf(`"%s" %s %d`, w.field, cond, v)
	}
	return ret
}

func (q *Query) String() string {
	sb := strings.Builder{}
	sb.WriteString(`SELECT "`)
	sb.WriteString(strings.Join(q.selects, `","`))
	sb.WriteString(`" FROM "`)

	if len(q.from.database) > 0 {
		sb.WriteString(q.from.database)
		sb.WriteString(`"."`)
	}
	sb.WriteString(q.from.table)
	sb.WriteString(`" FINAL `)

	if len(q.where) > 0 {
		sb.WriteString("WHERE (")
		conditions := make([]string, len(q.where))
		for idx, w := range q.where {
			conditions[idx] = w.string()
		}
		sb.WriteString(strings.Join(conditions, `) AND (`))
		sb.WriteString(`) `)
	}

	sb.WriteString(`ORDER BY _time `)
	if q.desc {
		sb.WriteString(`DESC `)
	} else {
		sb.WriteString(`ASC `)
	}

	sb.WriteString(`LIMIT `)
	sb.WriteString(strconv.Itoa(q.limit))

	return sb.String()
}

type MultiQuery struct {
	queries []*Query
	selects []string
	desc    bool
	limit   int
}

func NewMultiQuery() *MultiQuery {
	return &MultiQuery{
		limit: defaultQueryLimit,
	}
}

func (m *MultiQuery) Desc(desc bool) *MultiQuery {
	m.desc = desc
	return m
}

func (m *MultiQuery) Limit(limit int) *MultiQuery {
	m.limit = limit
	return m
}

func (m *MultiQuery) Join(queries []*Query) *MultiQuery {
	m.queries = append(m.queries, queries...)
	return m
}

func (m *MultiQuery) String() string {
	sbh := strings.Builder{}
	sb := strings.Builder{}
	alias := 0
	generateAlias := func() string {
		defer func() {
			alias++
		}()
		return "a" + strconv.Itoa(alias)
	}

	sbh.WriteString(`SELECT `)
	q0 := m.queries[0]
	q0.alias = generateAlias()
	sb.WriteString(q0.String())
	sb.WriteString(`) AS `)
	sb.WriteString(q0.alias)

	header := fmt.Sprintf(`", "%s"."`, q0.alias)
	sbh.WriteString(header[3:])
	sbh.WriteString(strings.Join(q0.selects, header))

	for i := 1; i < len(m.queries); i++ {
		qn := m.queries[i]
		sb.WriteString(` FULL OUTER JOIN (`)
		qn.alias = generateAlias()
		sb.WriteString(qn.String())
		sb.WriteString(fmt.Sprintf(`) AS %s ON %s._time = %s._time`, qn.alias, q0.alias, qn.alias))
		header = fmt.Sprintf(`", "%s"."`, qn.alias)
		sbh.WriteString(header)
		sbh.WriteString(strings.Join(qn.selects, header))
	}
	sbh.WriteString(`" `)

	sb.WriteString(` ORDER BY `)
	sb.WriteString(q0.alias)
	if m.desc {
		sb.WriteString(`._time  ASC `)
	} else {
		sb.WriteString(`._time  DESC `)
	}

	sb.WriteString(` LIMIT `)
	sb.WriteString(strconv.Itoa(m.limit))

	sbh.WriteString(` FROM (`)
	sbh.WriteString(sb.String())

	return sbh.String()
}
