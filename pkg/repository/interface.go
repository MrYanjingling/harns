package repository

type UpdateFn[T any] func(item *T) (T, error)

type Creator[C any, T any] interface {
	Create(ctx C, items []T) error
}

type Updater[C any, T any] interface {
	Update(ctx C, fn UpdateFn[T], filter Filter) (T, error)
}

type Clearer[C any, T any] interface {
	Clear(ctx C, filter Filter) error
}

type Finder[C any, T any] interface {
	Find(ctx C, query *Query) ([]T, error)
}

type Counter[C any, T any] interface {
	Count(ctx C, query *Query) (uint64, error)
}

type CRUDRepository[C any, T any] interface {
	Finder[C, T]
	Creator[C, T]
	Updater[C, T]
	Clearer[C, T]
	Counter[C, T]
}

type Repository[C any, T any] interface {
	CRUDRepository[C, T]
	Migrator[C, T]
	Watchable[C, T]
}

type Migrator[C any, T any] interface {
	Init(ctx C) error
	Migrate(ctx C, new Schema) error
}

type Watchable[C any, T any] interface {
	Watch(ctx C, filter Filter) (chan<- Mutation[T], error)
}

type Mutation[T any] struct {
	Add    *Add[T]
	Update *UpdateFn[T]
	Delete *Delete[T]
}

type Add[T any] []T
type Update[T any] struct {
	From T
	To   T
}
type Delete[T any] []T
