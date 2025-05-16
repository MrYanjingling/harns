package repository

import (
	"strings"
)

func newDialectSqlite() dialect {
	return dialectSqlite{}
}

type dialectSqlite struct{}

func (d dialectSqlite) buildCreateTable(table string, schema Schema) (string, error) {
	builder := strings.Builder{}

	builder.WriteString("CREATE TABLE IF NOT EXISTS")
	builder.WriteRune(' ')
	builder.WriteString(table)
	builder.WriteRune('(')

	properties, err := schema.Properties()

	if err != nil {
		return "", err
	}

	i := 0
	for key, property := range properties {
		if i > 0 {
			builder.WriteRune(',')
		}
		builder.WriteRune('\n')
		buildField(&builder, key, property)
		i++
	}

	builder.WriteRune(')')
	return builder.String(), nil
}

func buildField(builder *strings.Builder, name string, property Property) {
	builder.WriteString(name)
	builder.WriteRune(' ')

	switch property.Type().Name() {
	case TypeNameString:
		builder.WriteString("TEXT")
	case TypeNameNumber:
		builder.WriteString("REAL")
	case TypeNameInteger:
		builder.WriteString("INTEGER")
	case TypeNameBoolean:
		builder.WriteString("BOOLEAN")
	case TypeNameObject:
		builder.WriteString("JSON")
	case TypeNameArray:
		builder.WriteString("JSON")
	default:
		builder.WriteString("TEXT")
	}

	if name == "id" {
		builder.WriteRune(' ')
		builder.WriteString("PRIMARY KEY")
		if TypeNameInteger == property.Type().Name() && property.ReadOnly() {
			builder.WriteRune(' ')
			builder.WriteString("AUTOINCREMENT")
		}
		return
	}

	if !property.Nullable() {
		builder.WriteRune(' ')
		builder.WriteString("NOT NULL")
	}
}
