package repository

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type sqlRepo struct {
	sb sqlBuilder
	db *sql.DB
	tx *sql.Tx
}

func NewSqlite(dsn string) Repo {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(err)
	}
	return &sqlRepo{
		sb: sqlBuilder{
			dialect: dialectSqlite{},
		},
		db: db,
	}
}

var (
	_ Repo = (*sqlRepo)(nil)
)

func (s *sqlRepo) Find(ctx Context, query *Query) ([]Record, error) {
	properties, err := ctx.Schema.Properties()
	if err != nil {
		return nil, err
	}
	q, args, err := s.sb.buildFind(ctx.Name, ctx.Schema, query)
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

	objs := make([]Record, 0)
	for rows.Next() {
		obj := make(Object, len(columns))
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range columns {
			pointers[i] = &values[i]
		}
		err = rows.Scan(pointers...)
		if err != nil {
			return nil, err
		}
		for i, col := range columns {
			property, ok := properties[col]
			if !ok {
				continue
			}
			switch property.Type().Name() {
			case TypeNameObject:
				objField := make(Object)
				_ = objField.Scan(values[i])
				values[i] = objField
			case TypeNameArray:
				arrField := make(Array, 0)
				_ = arrField.Scan(values[i])
				values[i] = arrField
			default:
			}
			obj[col] = values[i]
		}
		objs = append(objs, obj)
	}

	if err = rows.Err(); err != nil {

		return nil, err
	}

	return objs, nil
}

func (s *sqlRepo) Create(ctx Context, items []Record) error {
	err := (error)(nil)
	q, args, err := s.sb.buildInsert(ctx.Name, ctx.Schema, items)
	if err != nil {
		return err
	}
	for i, arg := range args {
		if v, ok := arg.(map[string]any); ok {
			args[i] = Object(v)
			continue
		}
		if v, ok := arg.([]any); ok {
			args[i] = Array(v)
			continue
		}
	}
	rows, err := s.db.Query(q, args...)
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
		items[i].Set("id", id)
		i++
	}

	return nil
}

func (s *sqlRepo) Update(ctx Context, fn UpdateFn[Record], filter Filter) (Record, error) {
	find, err := s.Find(ctx, &Query{Filter: filter, Limit: 1})
	if err != nil || len(find) == 0 {
		return nil, err
	}
	old := find[0]

	fresh, err := fn(old.DeepCopy())
	if err != nil {
		return nil, err
	}

	q, args, err := s.sb.buildUpdate(ctx.Name, ctx.Schema, old, fresh, filter)
	if err != nil {
		return nil, err
	}

	res, err := s.db.Exec(q, args...)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrConflict
	}

	return fresh, nil
}

func (s *sqlRepo) Clear(ctx Context, filter Filter) error {
	err := (error)(nil)

	q, args, err := s.sb.buildClear(ctx.Name, ctx.Schema, filter)
	if err != nil {
		return err
	}
	result, err := s.db.Exec(q, args)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("")
	}
	return nil
}

func (s *sqlRepo) Count(ctx Context, query *Query) (uint64, error) {
	q, args, err := s.sb.buildCount(ctx.Name, ctx.Schema, query)
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

func (s *sqlRepo) Init(ctx Context) error {
	q, err := s.sb.buildCreateTable(ctx.Name, ctx.Schema)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(q)
	if err != nil {
		return err
	}
	return nil
}

func (s *sqlRepo) Migrate(ctx Context, new Schema) error {
	//TODO implement me
	return nil
}

func (s *sqlRepo) Watch(ctx Context, filter Filter) (chan<- Mutation[Record], error) {
	//TODO implement me
	return nil, nil
}
