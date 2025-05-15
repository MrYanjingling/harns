package repository

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
	MinLength *int
	MaxLength *int
	Pattern   *string
}

func (s *StringField) Name() TypeName {
	return TypeNameString
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

type IntegerField struct {
	Minimum          *int64
	Maximum          *int64
	ExclusiveMinimum *int64
	ExclusiveMaximum *int64
}

func (i *IntegerField) Name() TypeName {
	return TypeNameInteger
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
