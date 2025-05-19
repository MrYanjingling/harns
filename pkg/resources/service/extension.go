package service

import repo "lightiot/pkg/repository"

type BeforeInsert interface {
	BeforeInsert(schema repo.Schema, objs []repo.Object) error
}
type BeforeUpdate interface {
	BeforeUpdate(schema repo.Schema, old *repo.Object, new repo.Object) error
}
