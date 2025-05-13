package logstorage

import (
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
)

type Creater interface {
	Create(*data.TableDefinition) error
}

type Getter interface {
	Get(*query.Single) (interface{}, error)
}

type Lister interface {
	List(*query.Range) (interface{}, error)
	ListTables(p *query.Range) (interface{}, error)
}

type Inserter interface {
	Insert(*insert.Upsert) (interface{}, error)
}

type Updater interface {
	Update(*insert.Upsert) (interface{}, error)
}

type Deleter interface {
	Delete(*insert.Delete) (interface{}, error)
}

type Closer interface {
	Close()
}

type DataBase interface {
	GetTTLSeconds(database string) int
	GetTTLDays(database string) int
}

type Interface interface {
	Creater
	Getter
	Lister
	Inserter
	Updater
	Deleter
	Closer
	DataBase
}
