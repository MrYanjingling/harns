package repository

import (
	"database/sql"
)

type SqlRepo struct {
	sb SqlBuilder
}

type SqlContext struct {
	schema Schema
	table  string
	tx     *sql.Tx
}

func (s *SqlRepo) Find(ctx SqlContext, query *Query) ([]Object, error) {
	q, args, err := s.sb.buildFind(ctx.table, ctx.schema, query)
	if err != nil {
		return nil, err
	}

	rows, err := ctx.tx.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		panic(err)
	}

	var objs []Object
	for rows.Next() {
		obj := make(Object, len(columns))
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		err := rows.Scan(pointers...)
		if err != nil {
			return nil, err
		}
		for i, col := range columns {
			obj[col] = values[i]
		}
		objs = append(objs, obj)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return objs, nil
}

func (s *SqlRepo) Insert(ctx SqlContext, items []Object) error {
	q, args, err := s.sb.buildInsert(ctx.table, ctx.schema, items)
	if err != nil {
		return err
	}
	rows, err := ctx.tx.Query(q, args)
	if err != nil {
		return err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		id := new(any)
		err := rows.Scan(id)
		if err != nil {
			return err
		}
		items[i]["id"] = id
		i++
	}

	return nil
}

func (s *SqlRepo) Update(ctx SqlContext, fn UpdateFn[Object], filter Filter) ([]Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SqlRepo) Clear(ctx SqlContext, filter Filter) error {
	//TODO implement me
	panic("implement me")
}

func (s *SqlRepo) Count(ctx SqlContext, query *Query) (uint64, error) {
	//TODO implement me
	panic("implement me")
}
