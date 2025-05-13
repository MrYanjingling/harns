package repository

import (
	"context"
)

type UpdateFn[T any] func(item *T) T

type Repository[T any] interface {
	Find(ctx context.Context, query *Query) ([]T, error)
	Insert(ctx context.Context, items []T) error
	Update(ctx context.Context, fn UpdateFn[T], filters *Filters) (T, error)
	Clear(ctx context.Context, filters *Filters) error
	Count(ctx context.Context, query *Query) (int32, error)
}

type Migrator[T any] interface {
	Init(ctx context.Context) error
	Migrate(ctx context.Context) error
}
