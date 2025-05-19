package repository

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
)

// TODO: support filter by json filed
type dialect interface {
	buildCreateTable(table string, schema Schema) (string, error)
	parseJsonKey(key string) string
}

type sqlBuilder struct {
	dialect
}

func (s sqlBuilder) buildFind(table string, schema Schema, query *Query) (string, []any, error) {
	if query == nil {
		return "", nil, fmt.Errorf("query requires not null")
	}

	selects, err := s.selects(schema, query.Projection)
	if err != nil {
		return "", nil, err
	}

	builder := sq.Select(selects...).From(table)

	if query.Filter != nil {
		if exp, ok := query.Filter.(expression); ok {
			builder = builder.Where(exp.toSql(s.dialect))
		}
	}

	if query.Sort != nil {
		for field, order := range query.Sort {
			if order == SortOrderAsc {
				builder = builder.OrderBy(field)
			} else if order == SortOrderDesc {
				builder = builder.OrderBy(fmt.Sprintf("%s DESC", field))
			}
		}
	}

	if query.Skip > 0 {
		builder = builder.Offset(query.Skip)
	}
	if query.Limit > 0 {
		builder = builder.Limit(query.Limit)
	}
	if query.Distinct {
		builder = builder.Distinct()
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (s sqlBuilder) buildInsert(table string, schema Schema, objects []Object) (string, []any, error) {
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
		values := make([]any, 0, len(columns))
		for _, column := range columns {
			value := obj[column]
			values = append(values, value)
		}
		builder = builder.Values(values...)
	}
	builder = builder.Suffix("RETURNING 'id'")

	sql, args, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (s sqlBuilder) buildUpdate(table string, schema Schema, old Object, new Object, filter Filter) (string, []any, error) {
	builder := sq.Update(table)

	updates := make(map[string]any)
	for key, value := range new {
		if _, ok := old[key]; ok && !eq(old[key], value) {
			updates[key] = value
		}
	}

	if len(updates) == 0 {
		return "", nil, ErrNoFieldsToUpdate
	}

	builder = builder.SetMap(updates)

	if filter == nil {
		return "", nil, ErrInvalidFilter
	}

	if exp, ok := filter.(expression); ok {
		builder = builder.Where(exp.toSql(s.dialect))
	} else {
		return "", nil, ErrInvalidFilter
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (s sqlBuilder) buildClear(table string, schema Schema, filter Filter) (string, []any, error) {
	builder := sq.Delete(table)
	if filter == nil {
		return "", nil, ErrInvalidFilter
	}
	if exp, ok := filter.(expression); ok {
		builder = builder.Where(exp.toSql(s.dialect))
	} else {
		return "", nil, ErrInvalidFilter
	}
	return builder.ToSql()
}

func (s sqlBuilder) buildCount(table string, _ Schema, query *Query) (string, []any, error) {
	if query == nil {
		return "", nil, fmt.Errorf("query requires not null")
	}

	// TODO: support distinct
	builder := sq.Select("COUNT(*)").From(table)

	if query.Filter != nil {
		if exp, ok := query.Filter.(expression); ok {
			builder = builder.Where(exp.toSql(s.dialect))
		}
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (s sqlBuilder) selects(schema Schema, projection map[string]struct{}) ([]string, error) {
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
