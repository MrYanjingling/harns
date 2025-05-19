package resources

import repo "lightiot/pkg/repository"

type Resource struct {
	Table  string
	Schema repo.Schema
}

type ResourceName string
