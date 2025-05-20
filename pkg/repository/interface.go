package repository

type UpdateFn[T Record] func(item T) (T, error)

type Creator[C any, T Record] interface {
	Create(ctx C, items []T) error
}

type Updater[C any, T Record] interface {
	Update(ctx C, fn UpdateFn[T], filter Filter) (T, error)
}

type Clearer[C any, T Record] interface {
	Clear(ctx C, filter Filter) error
}

type Finder[C any, T Record] interface {
	Find(ctx C, query *Query) ([]T, error)
}

type Counter[C any, T Record] interface {
	Count(ctx C, query *Query) (uint64, error)
}

type CRUDRepository[C any, T Record] interface {
	Finder[C, T]
	Creator[C, T]
	Updater[C, T]
	Clearer[C, T]
	Counter[C, T]
}

type Repository[C any, T Record] interface {
	CRUDRepository[C, T]
	Migrator[C, T]
}

type Migrator[C any, T Record] interface {
	Init(ctx C) error
	Migrate(ctx C, new Schema) error
}

type Watchable[C any, T Record] interface {
	Watch(ctx C, filter Filter) (chan<- Mutation[T], error)
}

type Mutation[T Record] struct {
	Add    *Add[T]
	Update *UpdateFn[T]
	Delete *Delete[T]
}

type Add[T Record] []T
type Update[T Record] struct {
	From T
	To   T
}
type Delete[T Record] []T

type Repo Repository[Context, Record]
