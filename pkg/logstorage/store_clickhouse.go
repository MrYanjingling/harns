package logstorage

import (
	"database/sql"
	"fmt"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/logstorage/clickhouse"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"math"
	"time"
)

type ClickHouse struct {
	conn        *sql.DB
	clusterName string
	stopCh      <-chan struct{}
}

var _ Interface = new(ClickHouse)

var (
	datatype = []string{
		data.ColumnDatatypeString:    "String",
		data.ColumnDatatypeInt:       "Int32",
		data.ColumnDatatypeLong:      "Int64",
		data.ColumnDatatypeDouble:    "Float64",
		data.ColumnDatatypeBoolean:   "UInt8",
		data.ColumnDatatypeTimestamp: "DateTime64(6)",
	}
)

func convertDatatype(dt data.ColumnDatatype) string {
	return datatype[dt]
}

const (
	chSign         = "_sign"
	ChPartitionKey = "partition"
	ChEngineKey    = "engine"
	ChShardingKey  = "sharding"
)

func localTableName(name string) string {
	return name + "_local"
}

func allTableName(name string) string {
	return name + "_all"
}

func (ch *ClickHouse) createLocalTable(td *data.TableDefinition) error {
	cl := clickhouse.NewCreate()
	local := localTableName(td.GetTableName())
	db := td.GetTableProperty(data.Database).(string)

	cl.Cluster(ch.clusterName).
		Database(db).
		Table(local)

	for _, v := range td.GetColumns() {
		n, dt, ct := v.GetName(), v.GetDatatype(), v.GetColumnType()
		switch ct {
		case data.ColumnTypePrimaryKey:
			cl.PrimaryKey(n, convertDatatype(dt))
		case data.ColumnTypeSecondaryKey:
			cl.SecondaryKey(n, convertDatatype(dt))
		case data.ColumnTypeAttribute:
			// order _time can not be null
			if n == data.TimeKey {
				cl.Attribute(n, convertDatatype(dt), false)
			} else {
				cl.Attribute(n, convertDatatype(dt), true)
			}
		}
	}

	engine, _ := td.GetTableProperty(ChEngineKey).(clickhouse.EngineType)
	switch engine {
	case clickhouse.TypeReplicatedCollapsingMergeTree:
		cl.Engine(clickhouse.NewReplicatedCollapsingMergeTree(local, chSign))
	}

	if partitions, ok := td.GetTableProperty(ChPartitionKey).([]string); ok {
		cl.Partition(partitions)
	}

	if td.GetTTL() != math.MaxInt {
		cl.TTL(td.GetTTL())
	}

	if _, err := ch.conn.Exec(cl.String(), nil); err != nil {
		return err
	}

	return nil
}

func (ch *ClickHouse) createAllTable(td *data.TableDefinition) error {
	ca := clickhouse.NewCreate()
	local := localTableName(td.GetTableName())
	all := allTableName(td.GetTableName())
	db := td.GetTableProperty(data.Database).(string)

	ca.Cluster(ch.clusterName).
		Database(db).
		Table(all).
		AsTable(fmt.Sprintf(`"%s"."%s"`, db, local))

	shardingKey, _ := td.GetTableProperty(ChShardingKey).(string)
	ca.Engine(clickhouse.NewDistributed(ch.clusterName, db, local, shardingKey))

	if _, err := ch.conn.Exec(ca.String(), nil); err != nil {
		return err
	}

	return nil
}

func (ch *ClickHouse) Create(td *data.TableDefinition) (err error) {
	err = ch.createLocalTable(td)
	if err == nil {
		err = ch.createAllTable(td)
	}

	if err != nil {
		klog.V(3).InfoS("Failed to create", err)
	}

	return err
}

func hasColumn(columns map[string]*data.Column, column string) (ok bool) {
	_, ok = columns[column]
	return ok
}

func querySingleTable(
	database string,
	table *data.TableDefinition,
	filters map[string]interface{},
	selects []string,
	from, end time.Time,
	desc bool,
	limit int) (sb *clickhouse.Query) {

	tcs := table.GetColumns()
	q := clickhouse.NewQuery()
	qs := make([]string, 0, len(selects))
	qf := make([]*clickhouse.Where, 0, len(filters)+2) // +2 => _time: from end

	// if select exists in current table
	for _, s := range selects {
		if hasColumn(tcs, s) {
			qs = append(qs, s)
		}
	}

	// if filter exists in current table
	for k, f := range filters {
		if hasColumn(tcs, k) {
			qf = append(qf, clickhouse.Eq(k, f))
		}
	}

	qf = append(qf, []*clickhouse.Where{clickhouse.GE(data.TimeKey, from), clickhouse.LE(data.TimeKey, end)}...)

	q.Select(qs).
		From(database, allTableName(table.GetTableName())).
		Where(qf).
		Desc(desc).
		Limit(limit)

	return q
}

func (ch *ClickHouse) queryDB(
	query string,
	qs []*clickhouse.Query,
	tds query.TableDefinitions,
	limit int) (interface{}, error) {

	var err error
	defer func() {
		if err != nil {
			klog.V(3).InfoS("Failed to query", "err", err)
		}
	}()

	columnCnt := 0
	tables := tds.GetTableNames()
	boolIdx := make(map[int]struct{})
	timeIdx := make(map[int]struct{})
	skip := make([]int, len(tables))
	for i, table := range tables {
		// each table column start index
		skip[i] = columnCnt
		td := tds.GetTableDefinition(table)
		for _, s := range qs[i].Selects() {
			if s == data.TimeKey {
				timeIdx[columnCnt] = struct{}{}
			}
			if *td.GetColumnDatatype(s) == data.ColumnDatatypeBoolean {
				boolIdx[columnCnt] = struct{}{}
			}
			columnCnt++
		}
	}

	results := make([]interface{}, columnCnt)
	resultsPtr := make([]interface{}, columnCnt)
	for i := 0; i < columnCnt; i++ {
		resultsPtr[i] = &results[i]
	}

	ret := make([]map[string]interface{}, 0, columnCnt)

	rows, err := ch.conn.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var enough bool
	for rows.Next() {
		if err = rows.Scan(resultsPtr...); err != nil {
			return nil, err
		}

		idx := 0
		// split result to each table
		for _, q := range qs {
			retTmp := make(map[string]interface{}, len(q.Selects()))
			for j, s := range q.Selects() {
				bi := 0
				// invalid data
				if _, ok := timeIdx[idx]; ok && results[idx].(time.Time).Equal(time.Unix(0, 0)) {
					// skip current table
					if j+1 < len(q.Selects()) {
						idx = skip[j+1]
					}
					break
				} else if _, ok = boolIdx[idx]; ok {
					// convert bool
					if results[idx] == uint8(0) {
						results[idx] = false
					} else {
						results[idx] = true
					}
					bi++
				}

				retTmp[s] = results[idx]
				idx++
			}

			ret = append(ret, retTmp)
			if len(ret) == limit {
				enough = true
				break
			}
		}
		if enough {
			break
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (ch *ClickHouse) queryRange(
	database string,
	tables []string,
	tds query.TableDefinitions,
	filters map[string]interface{},
	selects []string,
	from, end time.Time,
	limit int,
	desc bool) (interface{}, error) {

	qs := make([]*clickhouse.Query, 0, len(tables))
	var queryString string
	if len(tables) == 1 {
		q := querySingleTable(
			database,
			tds.GetTableDefinition(tables[0]),
			filters,
			selects,
			from, end,
			desc,
			limit)
		qs = append(qs, q)
		queryString = q.String()
	} else {
		for _, table := range tables {
			q := querySingleTable(
				database,
				tds.GetTableDefinition(table),
				filters,
				selects,
				from, end,
				desc,
				limit)
			qs = append(qs, q)
		}

		qm := clickhouse.NewMultiQuery()
		qm.Join(qs).
			Desc(desc).
			Limit(limit)
		queryString = qm.String()
	}

	return ch.queryDB(queryString, qs, tds, limit)
}

func (ch *ClickHouse) Get(q *query.Single) (interface{}, error) {
	db := q.GetTableProperty(data.Database).(string)
	tables := q.GetTableNames()
	filters := q.GetKeys()
	selects := q.GetSelects().UnsortedList()
	end := time.Now().UTC()
	from := end.Add(-time.Duration(generic.GetTTLDay(q.GetTableDefinition(tables[0]).GetTTL())) * time.Hour * 24).UTC()
	limit := 2000
	if q.IsLatest() || len(filters) == 0 {
		limit = 1
	}
	return ch.queryRange(db, tables, q.TableDefinitions, filters, selects, from, end, limit, true)
}

func (ch *ClickHouse) List(r *query.Range) (interface{}, error) {
	db := r.GetTableProperty(data.Database).(string)
	tables := r.GetTableNames()
	filters := r.GetKeys()
	selects := r.GetSelects().UnsortedList()
	end := r.Range.GetEnd()
	from := r.Range.GetStart()
	return ch.queryRange(db, tables, r.TableDefinitions, filters, selects, from, end, r.GetLimit(), r.IsDesc())
}

func (ch *ClickHouse) ListTables(r *query.Range) (interface{}, error) {
	return nil, nil
}

func getColumnsValue(columns []string, row map[string]interface{}, values []interface{}) {
	for i := 0; i < len(columns); i++ {
		values[i] = row[columns[i]]
	}
}

func convertColumns(columns map[string]*data.Column) []string {
	ret := make([]string, 0, len(columns))
	for s, _ := range columns {
		ret = append(ret, s)
	}
	return ret
}

func (ch *ClickHouse) Insert(i *insert.Upsert) (interface{}, error) {
	var err error
	defer func() {
		if err != nil {
			klog.V(3).InfoS("Failed to insert", "err", err)
		}
	}()

	td := i.GetTableDefinition()
	cs := td.GetColumns()
	rows := i.GetRows()

	columns := make([]string, len(cs)+1)
	values := make([]interface{}, len(cs)+1) // +1 => _sign  depend on engine

	// insert all columns
	copy(columns, convertColumns(cs))
	columns[len(cs)] = chSign
	values[len(cs)] = 1

	ri := clickhouse.NewInsert()
	ri.Insert(i.GetTableProperty(data.Database).(string),
		allTableName(i.GetTableName())).
		Values(columns)

	tx, err := ch.conn.Begin()
	if err != nil {
		return nil, err
	}

	stmt, err := tx.Prepare(ri.String())
	if err != nil {
		return nil, err
	}

	for j := 0; j < len(rows); j++ {
		getColumnsValue(columns[:len(cs)], rows[j].GetValues(), values)
		if _, err = stmt.Exec(values...); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (ch *ClickHouse) Update(*insert.Upsert) (interface{}, error) {
	return nil, nil
}

func (ch *ClickHouse) Delete(*insert.Delete) (interface{}, error) {
	return nil, nil
}

func (ch *ClickHouse) Close() {
	if err := ch.conn.Close(); err != nil {
		klog.V(1).InfoS("Failed to close connection", "err", err)
	}
	ch.conn = nil
}

func (ch *ClickHouse) GetTTLSeconds(database string) int {
	return 0
}

func (ch *ClickHouse) GetTTLDays(database string) int {
	return 0
}
