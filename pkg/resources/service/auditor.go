package service

import (
	repo "lightiot/pkg/repository"
	"time"
)

type Auditor struct{}

func (a Auditor) BeforeInsert(schema repo.Schema, objs []repo.Object) error {
	property, err := schema.Property("createdTime")
	if err != nil {
		return nil
	}
	if *property.Format() != "date-time" {
		return nil
	}
	now := time.Now().Format("YYYY-MM-DDTHH:MM:SS.SSSZ")
	for _, obj := range objs {
		obj["createdTime"] = now
		obj["updatedTime"] = now
	}
	return nil
}

func (a Auditor) BeforeUpdate(schema repo.Schema, old *repo.Object, new repo.Object) error {
	property, err := schema.Property("updatedTime")
	if err != nil {
		return nil
	}
	if *property.Format() != "date-time" {
		return nil
	}
	now := time.Now().Format("YYYY-MM-DDTHH:MM:SS.SSSZ")
	new["updatedTime"] = now

	return nil
}
