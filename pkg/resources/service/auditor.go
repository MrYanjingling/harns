package service

import (
	repo "lightiot/pkg/repository"
	"time"
)

type Auditor struct{}

func (a Auditor) BeforeInsert(schema repo.Schema, objs []repo.Record) error {
	property, err := schema.Property("createdTime")
	if err != nil {
		return nil
	}
	if *property.Format() != "date-time" {
		return nil
	}
	now := time.Now().Format(time.RFC3339)
	for _, obj := range objs {
		obj.Set("createdTime", now)
		obj.Set("updatedTime", now)
	}
	return nil
}

func (a Auditor) BeforeUpdate(schema repo.Schema, old *repo.Record, new repo.Record) error {
	property, err := schema.Property("updatedTime")
	if err != nil {
		return nil
	}
	if *property.Format() != "date-time" {
		return nil
	}
	now := time.Now().Format("YYYY-MM-DDTHH:MM:SS.SSSZ")
	new.Set("updatedTime", now)

	return nil
}
