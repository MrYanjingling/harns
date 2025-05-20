package resources

import (
	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	"lightiot/pkg/apis/response"
	repo "lightiot/pkg/repository"
)

type ResourceName string
type Resource struct {
	Table    string
	Schema   repo.Schema
	Validate func(repo.Record) error
}

func NewResource(options []Option) (Resource, error) {
	resource := Resource{}
	for _, option := range options {
		err := option(&resource)
		if err != nil {
			return resource, err
		}
	}
	return resource, nil
}

type Option func(*Resource) error

func WithTable(table string) Option {
	return func(resource *Resource) error {
		resource.Table = table
		return nil
	}
}

func WithJsonSchema(jsonschema []byte) Option {
	return func(resource *Resource) error {
		schema, err := sonic.Get(jsonschema)

		if err != nil {
			return err
		}
		loader := gojsonschema.NewBytesLoader(jsonschema)
		validator, err := gojsonschema.NewSchema(loader)
		if err != nil {
			return err
		}
		validateFunc := func(record repo.Record) error {
			doc := gojsonschema.NewGoLoader(record)
			res, err := validator.Validate(doc)
			if err != nil {
				return err
			}
			if res.Valid() {
				return nil
			}

			errs := make([]error, 0, len(res.Errors()))
			for _, e := range res.Errors() {
				errs = append(errs, errors.Errorf("%s: %s", e.Field(), e.Description()))
			}

			return response.NewMultiError(errs...)
		}

		resource.Schema = repo.JsonSchema{
			Node: schema,
		}
		resource.Validate = validateFunc

		return nil
	}
}
