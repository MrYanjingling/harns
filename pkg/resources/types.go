package resources

import (
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/xeipuuv/gojsonschema"
	"lightiot/pkg/apis/response"
	repo "lightiot/pkg/repository"
	"strings"
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

			sb := strings.Builder{}

			for i, err := range res.Errors() {
				if i != 0 {
					sb.WriteRune(';')
					sb.WriteRune('\n')
				}
				sb.WriteString(err.String())
			}

			return response.ErrInvalidValue(fmt.Errorf(sb.String()))
		}

		resource.Schema = repo.JsonSchema{
			Node: schema,
		}
		resource.Validate = validateFunc

		return nil
	}
}
