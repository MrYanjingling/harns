package repository

import (
	"github.com/bytedance/sonic/ast"
)

type Schema interface {
	Properties() (map[string]Property, error)
	Property(name string) (Property, error)
	Title() *string
	Description() *string
	Required() ([]string, error)
}

type Property interface {
	Type() Type
	Default() any
	Description() *string
	Nullable() bool
	ReadOnly() bool
	Format() *string
}

type Type interface {
	Name() TypeName
}

type TypeName string

const (
	TypeNameString  TypeName = "String"
	TypeNameNumber  TypeName = "Number"
	TypeNameInteger TypeName = "Integer"
	TypeNameBoolean TypeName = "Boolean"
	TypeNameObject  TypeName = "Object"
	TypeNameArray   TypeName = "Array"
)

type StringField struct {
	MinLength *int64
	MaxLength *int64
	Pattern   *string
}

func (s *StringField) Name() TypeName {
	return TypeNameString
}

func (s *StringField) populate(node ast.Node) *StringField {
	minLength, err := node.Get("minLength").Int64()
	if err == nil {
		s.MinLength = &minLength
	}

	maxLength, err := node.Get("maxLength").Int64()
	if err == nil {
		s.MaxLength = &maxLength
	}

	pattern, err := node.Get("pattern").String()
	if err == nil {
		s.Pattern = &pattern
	}
	return s
}

type NumberField struct {
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum *float64
	ExclusiveMaximum *float64
}

func (n *NumberField) Name() TypeName {
	return TypeNameNumber
}

func (n *NumberField) populate(node ast.Node) *NumberField {
	minimum, err := node.Get("minimum").Float64()
	if err == nil {
		n.Minimum = &minimum
	}

	maximum, err := node.Get("maximum").Float64()
	if err == nil {
		n.Maximum = &maximum
	}

	exclusiveMinimum, err := node.Get("exclusiveMinimum").Float64()
	if err == nil {
		n.ExclusiveMinimum = &exclusiveMinimum
	}

	exclusiveMaximum, err := node.Get("exclusiveMaximum").Float64()
	if err == nil {
		n.ExclusiveMaximum = &exclusiveMaximum
	}
	return n
}

type IntegerField struct {
	Minimum          *int64
	Maximum          *int64
	ExclusiveMinimum *int64
	ExclusiveMaximum *int64
}

func (i *IntegerField) Name() TypeName {
	return TypeNameInteger
}

func (i *IntegerField) populate(node ast.Node) *IntegerField {
	minimum, err := node.Get("minimum").Int64()
	if err == nil {
		i.Minimum = &minimum
	}

	maximum, err := node.Get("maximum").Int64()
	if err == nil {
		i.Maximum = &maximum
	}

	exclusiveMinimum, err := node.Get("exclusiveMinimum").Int64()
	if err == nil {
		i.ExclusiveMinimum = &exclusiveMinimum
	}

	exclusiveMaximum, err := node.Get("exclusiveMaximum").Int64()
	if err == nil {
		i.ExclusiveMaximum = &exclusiveMaximum
	}
	return i
}

type BooleanField struct{}

func (b *BooleanField) Name() TypeName {
	return TypeNameBoolean
}

type ObjectField struct{}

func (o *ObjectField) Name() TypeName {
	return TypeNameObject
}

type ArrayField struct{}

func (a *ArrayField) Name() TypeName {
	return TypeNameArray
}

type JsonSchema struct {
	ast.Node
}

func (j JsonSchema) Properties() (map[string]Property, error) {
	node, err := j.Get("properties").MapUseNode()
	if err != nil {
		return nil, err
	}
	result := make(map[string]Property)
	for k, n := range node {
		result[k] = JsonProperty{n}
	}
	return result, nil
}

func (j JsonSchema) Property(name string) (Property, error) {
	node := j.Get("properties").Get(name)
	return JsonProperty{*node}, nil
}

func (j JsonSchema) Title() *string {
	s, err := j.Get("title").String()
	if err != nil {
		return new(string)
	}
	return &s
}

func (j JsonSchema) Description() *string {
	s, err := j.Get("description").String()
	if err != nil {
		return new(string)
	}
	return &s
}

func (j JsonSchema) Required() ([]string, error) {
	node, err := j.Get("required").ArrayUseNode()
	if err != nil {
		return nil, err
	}
	var result []string
	for _, n := range node {
		s, err := n.String()
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

type JsonProperty struct {
	ast.Node
}

func (j JsonProperty) Type() Type {
	typeName, err := j.Get("type").String()
	if err != nil {
		return nil
	}
	switch typeName {
	case "string":
		n := &StringField{}
		n.populate(j.Node)
		return n
	case "number":
		n := &NumberField{}
		n.populate(j.Node)
		return n
	case "integer":
		n := &IntegerField{}
		n.populate(j.Node)
		return n
	case "boolean":
		return &BooleanField{}
	case "object":
		return &ObjectField{}
	case "array":
		return &ArrayField{}
	default:
		return nil
	}
}

func (j JsonProperty) Default() any {
	defaultValue, err := j.Get("default").Interface()
	if err != nil {
		return nil
	}
	return defaultValue
}

func (j JsonProperty) Description() *string {
	s, err := j.Get("description").String()
	if err != nil {
		return new(string)
	}
	return &s
}

func (j JsonProperty) Nullable() bool {
	nullable, err := j.Get("nullable").Bool()
	if err != nil {
		return false
	}
	return nullable
}

func (j JsonProperty) ReadOnly() bool {
	readOnly, err := j.Get("readOnly").Bool()
	if err != nil {
		return false
	}
	return readOnly
}

func (j JsonProperty) Format() *string {
	s, err := j.Get("format").String()
	if err != nil {
		return new(string)
	}
	return &s
}
