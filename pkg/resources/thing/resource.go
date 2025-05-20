package thing

import (
	_ "embed"
	res "lightiot/pkg/resources"
)

//go:embed schema.json
var jsonschema []byte

func Resource() (res.ResourceName, res.Resource) {
	options := []res.Option{
		res.WithTable("thing"),
		res.WithJsonSchema(jsonschema),
	}
	resource, err := res.NewResource(options)
	if err != nil {
		panic(err)
	}
	return "things", resource
}
