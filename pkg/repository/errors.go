package repository

import "errors"

var (
	ErrConflict         = errors.New("conflict")
	ErrInvalidFilter    = errors.New("invalid filter")
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)
