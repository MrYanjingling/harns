package repository

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
)

type Dialect interface {
}

type SqlBuilder struct {
	Dialect
}

func (s SqlBuilder) buildFind(table string, schema Schema, query *Query) (string, []any, error) {
	selects, err := s.selects(schema, query.Projection)
	if err != nil {
		return "", nil, err
	}

	builder := sq.Select(selects...).From(table)

	builder = s.applyFilter(query.Filter, builder)

	if query.Sort != nil {
		for field, order := range query.Sort {
			if order == SortOrderAsc {
				builder = builder.OrderBy(field)
			} else if order == SortOrderDesc {
				builder = builder.OrderBy(fmt.Sprintf("%s DESC", field))
			}
		}
	}

	if query.Skip != nil {
		builder.Offset(uint64(*query.Skip))
	}
	if query.Limit != nil {
		builder.Limit(uint64(*query.Limit))
	}
	if query.Distinct {
		builder.Distinct()
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (s SqlBuilder) buildInsert(table string, schema Schema, objects []Object) (string, []any, error) {
	if len(objects) == 0 {
		return "", nil, fmt.Errorf("no objects provided for insertion")
	}

	properties, err := schema.Properties()
	if err != nil {
		return "", nil, err
	}

	columns := make([]string, 0, len(properties))

	for key := range properties {
		columns = append(columns, key)
	}

	builder := sq.Insert(table).Columns(columns...)

	for _, obj := range objects {
		values := make([]any, 0, len(properties))
		for key := range properties {
			value := obj[key]
			values = append(values, value)
		}
		builder = builder.Values(values...)
	}
	builder.Suffix("RETURNING 'id'")

	sql, args, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (s SqlBuilder) applyFilter(filter Filter, builder sq.SelectBuilder) sq.SelectBuilder {
	if filter != nil {
		if exp, ok := filter.(expression); ok {
			builder = builder.Where(exp.toExp())
		}
	}
	return builder
}

func (s SqlBuilder) selects(schema Schema, projection map[string]bool) ([]string, error) {
	properties, err := schema.Properties()
	if err != nil {
		return nil, err
	}

	selects := make([]string, 0, 16)

	if len(projection) == 0 {
		for key := range properties {
			selects = append(selects, key)
		}
		return selects, nil
	}

	for key := range projection {
		if _, ok := properties[key]; ok {
			selects = append(selects, key)
		}
	}
	return selects, nil
}
