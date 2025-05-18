package repository

import (
	"database/sql"
	"fmt"
)

type Repo struct {
	sb sqlBuilder
	db *sql.DB
	tx *sql.Tx
}

var (
	_ Repository[Ctx, Object] = (*Repo)(nil)
)

func (s *Repo) Find(ctx Ctx, query *Query) ([]Object, error) {
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

	objs := make([]Object, 0)
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

func (s *Repo) Insert(ctx Ctx, items []Object) error {
	err := (error)(nil)
	tx := s.tx
	if tx == nil {
		tx, _ = s.db.Begin()
		defer rollbackIfErrFunc(err, tx)()
	}

	q, args, err := s.sb.buildInsert(ctx.table, ctx.schema, items)
	if err != nil {
		return err
	}
	rows, err := tx.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		id := new(any)
		err = rows.Scan(id)
		if err != nil {
			return err
		}
		items[i]["id"] = id
		i++
	}

	return nil
}

func rollbackIfErrFunc(err error, tx *sql.Tx) func() {
	return func() {
		if err == nil {
			_ = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}
}

func (s *Repo) Update(ctx Ctx, fn UpdateFn[Object], filter Filter) (Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Repo) Clear(ctx Ctx, filter Filter) error {
	err := (error)(nil)
	tx := s.tx
	if tx == nil {
		tx, _ = s.db.Begin()
		defer rollbackIfErrFunc(err, tx)()
	}

	q, args, err := s.sb.buildClear(ctx.table, ctx.schema, filter)
	if err != nil {
		return err
	}
	result, err := tx.Exec(q, args)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("")
	}
	return nil
}

func (s *Repo) Count(ctx Ctx, query *Query) (uint64, error) {
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

func (s *Repo) Init(ctx Ctx) error {
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

func (s *Repo) Migrate(ctx Ctx, new Schema) error {
	//TODO implement me
	panic("implement me")
}
