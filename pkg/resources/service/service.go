package service

import (
	"fmt"
	repo "lightiot/pkg/repository"
	res "lightiot/pkg/resources"
)

type Service struct {
	resources     map[res.ResourceName]res.Resource
	repo          repo.Repository[repo.Context, repo.Record]
	beforeInserts []BeforeInsert
	beforeUpdates []BeforeUpdate
	watchers      map[res.ResourceName]map[repo.Filter]chan<- repo.Mutation[repo.Record]
}

func New(repo repo.Repository[repo.Context, repo.Record]) Service {
	return Service{
		resources:     make(map[res.ResourceName]res.Resource),
		repo:          repo,
		beforeInserts: []BeforeInsert{Auditor{}},
		beforeUpdates: []BeforeUpdate{Auditor{}},
	}
}

func (s *Service) AddResource(name res.ResourceName, resource res.Resource) {
	s.resources[name] = resource
	err := s.Init(name)
	if err != nil {
		panic(err)
	}
}

var (
	_ repo.Repository[res.ResourceName, repo.Record] = (*Service)(nil)
)

func (s *Service) Find(ctx res.ResourceName, query *repo.Query) ([]repo.Record, error) {
	repoCtx, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.Find(repoCtx, query)
}

func (s *Service) Create(ctx res.ResourceName, items []repo.Record) error {
	repoCtx, err := s.getContext(ctx)
	if err != nil {
		return err
	}
	for _, fn := range s.beforeInserts {
		err = fn.BeforeInsert(repoCtx.Schema, items)
		if err != nil {
			return err
		}
	}

	return s.repo.Create(repoCtx, items)
}

func (s *Service) Update(ctx res.ResourceName, fn repo.UpdateFn[repo.Record], filter repo.Filter) (repo.Record, error) {
	repoCtx, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}

	mutate := func(old *repo.Record) (repo.Record, error) {
		fresh, e := fn(old)

		for _, bu := range s.beforeUpdates {
			e = bu.BeforeUpdate(repoCtx.Schema, old, fresh)
			if e != nil {
				return nil, e
			}
		}
		return fresh, nil
	}

	return s.repo.Update(repoCtx, mutate, filter)
}

func (s *Service) Clear(ctx res.ResourceName, filter repo.Filter) error {
	repoCtx, err := s.getContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.Clear(repoCtx, filter)
}

func (s *Service) Count(ctx res.ResourceName, query *repo.Query) (uint64, error) {
	repoCtx, err := s.getContext(ctx)
	if err != nil {
		return 0, err
	}

	return s.repo.Count(repoCtx, query)
}

func (s *Service) getContext(ctx res.ResourceName) (repo.Context, error) {
	resource, ok := s.resources[ctx]
	if !ok {
		return repo.Context{}, fmt.Errorf("no such resource: %s", resource)
	}
	return repo.Context{
		Name:   resource.Table,
		Schema: resource.Schema,
	}, nil
}

func (s *Service) Init(ctx res.ResourceName) error {
	repoCtx, err := s.getContext(ctx)
	if err != nil {
		return err
	}
	return s.repo.Init(repoCtx)
}

func (s *Service) Migrate(ctx res.ResourceName, new repo.Schema) error {
	//TODO implement me
	panic("implement me")
}

func (s *Service) Watch(ctx res.ResourceName, filter repo.Filter) (chan<- repo.Mutation[repo.Record], error) {
	ch := make(chan repo.Mutation[repo.Record], 1)

	return ch, nil
}
