package service

import repo "lightiot/pkg/repository"

type BeforeInsert interface {
	BeforeInsert(schema repo.Schema, records []repo.Record) error
}
type BeforeUpdate interface {
	BeforeUpdate(schema repo.Schema, old repo.Record, new repo.Record) error
}
