package clickhouse

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type EngineType byte

const (
	TypeDistributed EngineType = iota
	TypeReplicatedCollapsingMergeTree
)

type engineer interface {
	inject() string
	engine() string
	setting() string
	engineType() EngineType
}

type Distributed struct {
	cluster  string
	database string
	table    string
	key      string
}

func NewDistributed(cluster, database, table, key string) *Distributed {
	return &Distributed{
		cluster:  cluster,
		database: database,
		table:    table,
		key:      key,
	}
}

func (d *Distributed) inject() string {
	return ""
}

func (d *Distributed) engine() string {
	return fmt.Sprintf(` ENGINE = Distributed("%s", "%s", "%s", halfMD5("%s")) `, d.cluster, d.database, d.table, d.key)
}

func (d *Distributed) setting() string {
	return ""
}

func (d *Distributed) engineType() EngineType {
	return TypeDistributed
}

type ReplicatedCollapsingMergeTree struct {
	table string
	ver   string
}

func NewReplicatedCollapsingMergeTree(table, ver string) *ReplicatedCollapsingMergeTree {

	return &ReplicatedCollapsingMergeTree{
		table: table,
		ver:   ver,
	}
}

func (r *ReplicatedCollapsingMergeTree) inject() string {
	return fmt.Sprintf(`"%s" Int8, `, r.ver)
}

func (r *ReplicatedCollapsingMergeTree) engine() string {
	return fmt.Sprintf(` ENGINE = ReplicatedCollapsingMergeTree('/clickhouse/tables/{shard}/%s', '{replica}', "%s") `, r.table, r.ver)
}

func (r *ReplicatedCollapsingMergeTree) setting() string {
	// TODO get setting from configmap
	return "SETTINGS min_rows_for_compact_part = 100, merge_with_ttl_timeout = 10"
}

func (r *ReplicatedCollapsingMergeTree) engineType() EngineType {
	return TypeReplicatedCollapsingMergeTree
}

type Create struct {
	engine   engineer
	cluster  string
	database string
	table    string
	asTable  string
	ttl      int
	pk       []*struct {
		field    string
		datatype string
	}
	sk  []string
	att []*struct {
		field    string
		datatype string
		nullable bool
	}
	partitions []string
	settings   []string
}

func NewCreate() *Create {
	return &Create{}
}

func (c *Create) Database(db string) *Create {
	c.database = db
	return c
}

func (c *Create) Table(table string) *Create {
	c.table = table
	return c
}

func (c *Create) Cluster(cluster string) *Create {
	c.cluster = cluster
	return c
}

func (c *Create) Partition(partitions []string) *Create {
	c.partitions = append(c.partitions, partitions...)
	return c
}

func (c *Create) TTL(ttl int) *Create {
	c.ttl = ttl
	return c
}

func (c *Create) Attribute(name string, dt string, nullable bool) *Create {
	t := &struct {
		field    string
		datatype string
		nullable bool
	}{

		field:    name,
		datatype: dt,
		nullable: nullable,
	}
	c.att = append(c.att, t)
	return c
}

func (c *Create) PrimaryKey(name string, dt string) *Create {
	t := &struct {
		field    string
		datatype string
	}{
		field:    name,
		datatype: dt,
	}
	c.pk = append(c.pk, t)
	return c
}

func (c *Create) SecondaryKey(name string, dt string) *Create {
	c.Attribute(name, dt, false)
	c.sk = append(c.sk, name)
	return c
}

func (c *Create) AsTable(s string) *Create {
	c.asTable = s
	return c
}

func (c *Create) Engine(engine engineer) *Create {
	c.engine = engine
	return c
}

func (c *Create) String() string {
	buf := &bytes.Buffer{}

	buf.WriteString(`CREATE TABLE IF NOT EXISTS "`)

	if len(c.database) > 0 {
		buf.WriteString(c.database)
		buf.WriteString(`"."`)
	}

	buf.WriteString(c.table)
	buf.WriteRune('"')

	if len(c.cluster) > 0 {
		buf.WriteString(` ON CLUSTER "`)
		buf.WriteString(c.cluster)
		buf.WriteRune('"')
	}

	if len(c.asTable) > 0 {
		buf.WriteString(" AS ")
		buf.WriteString(c.asTable)
	} else {
		buf.WriteString(` (`)

		buf.WriteString(c.engine.inject())

		for _, pk := range c.pk {
			buf.WriteString(`"`)
			buf.WriteString(pk.field)
			buf.WriteString(`" `)
			buf.WriteString(pk.datatype)
			if len(c.att) != 0 {
				buf.WriteString(`, `)
			}
		}

		for _, att := range c.att {
			buf.WriteString(`"`)
			buf.WriteString(att.field)
			buf.WriteString(`" `)
			if att.nullable {
				buf.WriteString(`Nullable(`)
				buf.WriteString(att.datatype)
				buf.WriteString(`)`)
			} else {
				buf.WriteString(att.datatype)
			}
			if len(c.sk) != 0 {
				buf.WriteString(`, `)
			}
		}

		for i, sk := range c.sk {
			buf.WriteString(`INDEX "`)
			buf.WriteString(sk)
			buf.WriteString(`_idx" "`)
			buf.WriteString(sk)
			buf.WriteString(`" TYPE set(0) GRANULARITY 1`)
			if i < len(c.sk)-1 {
				buf.WriteString(`, `)
			}
		}

		buf.WriteString(` )`)
	}

	buf.WriteString(c.engine.engine())

	if c.engine.engineType() != TypeDistributed {
		// PARTITION BY
		if len(c.partitions) > 0 {
			buf.WriteString(` PARTITION BY ("`)
			buf.WriteString(strings.Join(c.partitions, `", "`))
			buf.WriteString(`")`)
		}

		// ORDER BY
		buf.WriteString(` ORDER BY ("`)
		// It is possible to specify a primary key that is different from the sorting key .
		// In this case the primary key expression tuple must be a prefix of the sorting key expression tuple.
		// https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree/#:~:text=In%20this%20case%20the%20primary%20key%20expression%20tuple%20must%20be%20a%20prefix%20of%20the%20sorting%20key%20expression%20tuple
		for _, pk := range c.pk {
			buf.WriteString(pk.field)
			buf.WriteString(`", "`)
		}
		buf.WriteString(`_time" )`)

		// TTL
		if c.ttl > 0 {
			buf.WriteString(` TTL toDateTime(_time) + toIntervalDay(`)
			buf.WriteString(strconv.Itoa(c.ttl))
			buf.WriteString(`) `)
		}

	}

	// SETTING
	buf.WriteString(c.engine.setting())

	return buf.String()
}
