package repository

type UpdateFn[T any] func(item *T) T

type Repository[C any, T any] interface {
	Find(ctx C, query *Query) ([]T, error)
	Insert(ctx C, items []T) error
	Update(ctx C, fn UpdateFn[T], filter Filter) ([]T, error)
	Clear(ctx C, filter Filter) error
	Count(ctx C, query *Query) (uint64, error)
}

type Migrator[T any, C any] interface {
	Init(ctx C) error
	Migrate(ctx C) error
}
