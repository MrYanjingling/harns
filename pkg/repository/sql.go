package repository

import (
	"database/sql"
)

type SqlRepo struct {
	SqlBuilder
}

type SqlContext struct {
	Schema
	table string
	tx    *sql.Tx
}

func (s SqlRepo) Find(ctx SqlContext, query *Query) ([]Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s SqlRepo) Insert(ctx SqlContext, items []Object) error {
	//TODO implement me
	panic("implement me")
}

func (s SqlRepo) Update(ctx SqlContext, fn UpdateFn[Object], filter Filter) ([]Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s SqlRepo) Clear(ctx SqlContext, filter Filter) error {
	//TODO implement me
	panic("implement me")
}

func (s SqlRepo) Count(ctx SqlContext, query *Query) (uint64, error) {
	//TODO implement me
	panic("implement me")
}
