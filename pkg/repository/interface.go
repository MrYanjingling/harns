package repository

type UpdateFn[T any] func(item *T) T

type Repository[T any, C any] interface {
	Find(ctx C, query *Query) ([]T, error)
	Insert(ctx C, items []T) error
	Update(ctx C, fn UpdateFn[T], filters *Filters) ([]T, error)
	Clear(ctx C, filters *Filters) error
	Count(ctx C, query *Query) (uint64, error)
}

type Migrator[T any, C any] interface {
	Init(ctx C) error
	Migrate(ctx C) error
}
