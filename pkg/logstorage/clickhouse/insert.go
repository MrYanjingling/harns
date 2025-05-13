package clickhouse

import "strings"

type Insert struct {
	into struct {
		database string
		table    string
	}
	values []string
}

/*
INSERT INTO "default"."StandardEvent_all" ("correlationId", "thingId", "_time", "severity", "code", "source", "acknowledged", "id", "_sign") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
*/

func NewInsert() *Insert {
	return &Insert{}
}

func (i *Insert) Insert(database, table string) *Insert {
	i.into.database = database
	i.into.table = table
	return i
}

func (i *Insert) Values(values []string) *Insert {
	i.values = append(i.values, values...)
	return i
}

/*
INSERT INTO "default"."StandardEvent_all" ("correlationId", "thingId", "_time", "severity", "code", "source", "acknowledged", "id", "_sign") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
*/
func (i *Insert) String() string {
	sb := strings.Builder{}

	sb.WriteString(`INSERT INTO "`)
	if len(i.into.database) > 0 {
		sb.WriteString(i.into.database)
		sb.WriteString(`"."`)
	}
	sb.WriteString(i.into.table)
	sb.WriteString(`" ("`)
	sb.WriteString(strings.Join(i.values, `", "`))
	sb.WriteString(`") VALUES (?`)

	for j := 1; j < len(i.values); j++ {
		sb.WriteString(`, ?`)
	}
	sb.WriteString(`)`)

	return sb.String()
}
