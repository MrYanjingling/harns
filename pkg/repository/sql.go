package repository

import (
	"database/sql"
	"fmt"
)

type SqlRepo struct {
	sb sqlBuilder
	db *sql.DB
}

type SqlContext struct {
	schema Schema
	table  string
	tx     *sql.Tx
}

func (s *SqlRepo) Find(ctx SqlContext, query *Query) ([]Object, error) {
	properties, err := ctx.schema.Properties()
	if err != nil {
		return nil, err
	}
	q, args, err := s.sb.buildFind(ctx.table, ctx.schema, query)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(q, args...)
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
		for i, column := range columns {
			property, ok := properties[column]
			if !ok {
				continue
			}
			switch property.Type().Name() {
			case TypeNameObject:
				values[i] = make(Object)
			case TypeNameArray:
				values[i] = make(Array, 0)
			default:
			}
			pointers[i] = &values[i]
		}
		err = rows.Scan(pointers...)
		if err != nil {
			return nil, err
		}
		for i, col := range columns {
			obj[col] = values[i]
		}
		objs = append(objs, obj)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return objs, nil
}

func (s *SqlRepo) Insert(ctx SqlContext, items []Object) error {
	tx := ctx.tx
	if tx == nil {
		tx, _ = s.db.Begin()
	}

	q, args, err := s.sb.buildInsert(ctx.table, ctx.schema, items)
	if err != nil {
		return err
	}
	rows, err := tx.Query(q, args...)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		id := new(any)
		err := rows.Scan(id)
		if err != nil {
			tx.Rollback()
			return err
		}
		items[i]["id"] = id
		i++
	}

	tx.Commit()

	return nil
}

func (s *SqlRepo) Update(ctx SqlContext, fn UpdateFn[Object], filter Filter) (Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SqlRepo) Clear(ctx SqlContext, filter Filter) error {
	tx := ctx.tx
	if tx == nil {
		tx, _ = s.db.Begin()
	}

	q, args, err := s.sb.buildClear(ctx.table, ctx.schema, filter)
	if err != nil {
		return err
	}
	result, err := tx.Exec(q, args)
	if err != nil {
		tx.Rollback()
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		return fmt.Errorf("")
	}
	tx.Commit()
	return nil
}

func (s *SqlRepo) Count(ctx SqlContext, query *Query) (uint64, error) {
	q, args, err := s.sb.buildCount(ctx.table, ctx.schema, query)
	if err != nil {
		return 0, err
	}
	row := s.db.QueryRow(q, args)

	if row.Err() != nil {
		return 0, row.Err()
	}
	count := new(uint64)
	err = row.Scan(count)
	if err != nil {
		return 0, err
	}
	return *count, nil
}

func (s *SqlRepo) Init(ctx SqlContext) error {
	q, err := s.sb.buildCreateTable(ctx.table, ctx.schema)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(q)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlRepo) Migrate(ctx SqlContext, new Schema) error {
	//TODO implement me
	panic("implement me")
}
