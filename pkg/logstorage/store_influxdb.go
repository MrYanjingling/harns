package logstorage

import (
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	fluxquery "github.com/influxdata/influxdb-client-go/v2/api/query"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"time"
)

// Do NOT support query cross retention policies

const (
	org           = "main"
	secondsPerDay = 24 * 60 * 60
	// https://docs.influxdata.com/influxdb/v2.1/api/#operation/PostBuckets
	errCodeExists              = "conflict"
	TasksSystemBucketName      = "_tasks"
	MonitoringSystemBucketName = "_monitoring"
)

type InfluxDB struct {
	token     string
	client    influxdb2.Client
	org       *domain.Organization
	stopCh    <-chan struct{}
	databases map[string]data.DatabaseInfo
}

var _ Interface = new(InfluxDB)

func NewInfluxDB(stopCh <-chan struct{}, url, token string) (Interface, error) {
	opt := influxdb2.DefaultOptions()
	opt.SetPrecision(time.Millisecond)

	c := influxdb2.NewClientWithOptions(url, token, opt)

	if _, err := c.Ready(context.Background()); err != nil {
		klog.InfoS("Not ready", "err", err)
		return nil, err
	}

	o, err := c.OrganizationsAPI().FindOrganizationByName(context.Background(), org)
	if err != nil {
		klog.InfoS("Failed to find organization", "err", err)
		return nil, err
	}
	if o == nil {
		klog.InfoS("Organization not found", "err", err)
		return nil, err
	}

	buckets, err := c.BucketsAPI().GetBuckets(context.Background(), api.PagingWithLimit(20))
	if err != nil {
		klog.InfoS("Failed to get databases", "err", err)
		return nil, err
	}
	if buckets == nil {
		klog.InfoS("databases not found", "err", err)
		return nil, err
	}
	databases := make(map[string]data.DatabaseInfo)
	for _, bucket := range *buckets {
		if bucket.Name == TasksSystemBucketName || bucket.Name == MonitoringSystemBucketName {
			continue
		}
		ttl := bucket.RetentionRules[0].EverySeconds
		databases[bucket.Name] = data.NewDataBaseInfo(bucket.Name, ttl)
	}

	return &InfluxDB{
		token:     token,
		client:    c,
		org:       o,
		stopCh:    stopCh,
		databases: databases,
	}, nil
}

func (influx *InfluxDB) Create(td *data.TableDefinition) error {
	return nil
}

func (influx *InfluxDB) Get(stmt *query.Single) (interface{}, error) {
	bucket := stmt.GetTableProperty(data.Database).(string)
	ttl := stmt.GetTTL()

	queryTags := ""
	for _, n := range stmt.GetTableNames() {
		queryTags += fmt.Sprintf(`r._measurement == "%s" and`, n)
	}

	queryTags = queryTags[:len(queryTags)-4]

	queryFilter := ""
	for k, v := range stmt.GetKeys() {
		if stmt.TableDefinitions.IsPrimaryKey(k) {
			queryTags += fmt.Sprintf(` and r["%s"] == "%v"`, k, v)
		} else if stmt.TableDefinitions.IsSecondaryKey(k) {
			switch stmt.GetColumnDatatype(k) {
			case data.ColumnDatatypeString:
				queryFilter += fmt.Sprintf(` r["%s"] == "%v" and`, k, v)
			default:
				queryFilter += fmt.Sprintf(` r["%s"] == %v and`, k, v)
			}
		}
	}
	if len(queryFilter) != 0 {
		queryFilter = queryFilter[:len(queryFilter)-4]
	}

	queryFields := ""
	for i, f := range stmt.GetSelects().UnsortedList() {
		if i != 0 {
			queryFields += " or"
		}
		queryFields += fmt.Sprintf(` r._field == "%s"`, f)
	}

	var queryString string

	if stmt.IsLatest() || len(queryFilter) == 0 {

		if len(queryFields) == 0 {
			queryFields = "true"
		}

		queryTemplate := `from(bucket:"%s%s")
|> range(start: %s, stop: %s)
|> filter(fn: (r) => %s)
|> filter(fn: (r) => %s)
%s
|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
`

		if len(queryFilter) != 0 {
			queryTemplate += fmt.Sprintf("|> filter(fn: (r) => %s)\n", queryFilter)
		}

		var limit string
		if stmt.IsLatest() {
			limit += `|> last()`
		} else if len(queryFilter) == 0 {
			limit += `|> limit(n:1)`
		}

		queryString = fmt.Sprintf(queryTemplate, bucket, generic.GetRPName(ttl), fmt.Sprintf("-%dd", generic.GetTTLDay(ttl)), stmt.GetEnd().Format(time.RFC3339Nano), queryTags, queryFields, limit)
	} else {
		keys := sets.String{}
		for key, _ := range stmt.GetKeys() {
			keys.Insert(key)
		}
		diff := keys.Difference(stmt.GetSelects()).UnsortedList()
		for _, key := range diff {
			if len(queryFields) > 0 {
				queryFields += " or"
			}
			queryFields += fmt.Sprintf(` r._field == "%s"`, key)
		}

		queryTemplate := `from(bucket:"%s%s")
|> range(start: %s, stop: %s)
|> filter(fn: (r) => %s)
|> filter(fn: (r) => %s)
|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
|> filter(fn: (r) => %s)
|> limit(n:1)`
		queryString = fmt.Sprintf(queryTemplate, bucket, generic.GetRPName(stmt.GetTTL()), fmt.Sprintf("-%dd", generic.GetTTLDay(ttl)), stmt.GetEnd().Format(time.RFC3339Nano), queryTags, queryFields, queryFilter)
	}

	result, err := influx.client.QueryAPI(influx.org.Name).Query(context.Background(), queryString)
	ret := make([]map[string]interface{}, 0)
	if err == nil {
		for result.Next() {
			if result.TableChanged() {
				klog.V(6).InfoS("Table changed", "table", result.TableMetadata().String())
			}
			klog.V(6).InfoS("Row", "row", result.Record().String())

			item := map[string]interface{}{
				data.TimeKey: result.Record().Time(),
			}
			for k, v := range result.Record().Values() {
				if stmt.GetSelects().Has(k) {
					item[k] = v
				}
			}
			ret = append(ret, item)
		}
		if result.Err() != nil {
			klog.V(3).InfoS("Failed to query", "err", result.Err().Error())
		}
	} else {
		klog.V(3).InfoS("Failed to query", "err", err.Error())
	}

	return ret, nil
}

func (influx *InfluxDB) List(stmt *query.Range) (interface{}, error) {
	bucket := stmt.GetTableProperty(data.Database).(string)

	queryTags := ""
	for _, n := range stmt.GetTableNames() {
		queryTags += fmt.Sprintf(` r._measurement == "%s" or`, n)
	}

	queryTags = queryTags[:len(queryTags)-3]

	queryFilter := ""
	for k, v := range stmt.GetKeys() {
		if stmt.TableDefinitions.IsPrimaryKey(k) {
			queryTags += fmt.Sprintf(` and r["%s"] == "%v"`, k, v)
		} else if stmt.TableDefinitions.IsSecondaryKey(k) {
			switch stmt.GetColumnDatatype(k) {
			case data.ColumnDatatypeString:
				queryFilter += fmt.Sprintf(` r["%s"] == "%v" and`, k, v)
			default:
				queryFilter += fmt.Sprintf(` r["%s"] == %v and`, k, v)
			}
		}
	}
	if stmt.ByTime() {
		povit := `|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`
		if len(queryFilter) != 0 {
			queryFilter = queryFilter[:len(queryFilter)-4]
			queryFilter = fmt.Sprintf(`%s
|> filter(fn: (r) => %s)`, povit, queryFilter)
		} else {
			queryFilter = povit
		}
	}

	queryFields := ""
	for i, f := range stmt.GetSelects().UnsortedList() {
		if i != 0 {
			queryFields += " or"
		}
		queryFields += fmt.Sprintf(` r._field == "%s"`, f)
	}

	keys := sets.String{}
	for key, _ := range stmt.GetKeys() {
		keys.Insert(key)
	}
	diff := keys.Difference(stmt.GetSelects()).UnsortedList()
	for _, key := range diff {
		if len(queryFields) > 0 {
			queryFields += " or"
		}
		queryFields += fmt.Sprintf(` r._field == "%s"`, key)
	}

	var queryString string
	queryTemplate := `from(bucket:"%s%s")
|> range(start: %s, stop: %s)
|> filter(fn: (r) => %s)
|> filter(fn: (r) => %s)
%s
|> sort(columns:["_time"], desc: %t)
|> limit(n:%d)`

	queryString = fmt.Sprintf(queryTemplate, bucket, generic.GetRPName(stmt.GetTTL()), stmt.GetStart().Format(time.RFC3339Nano), stmt.GetEnd().Format(time.RFC3339Nano), queryTags, queryFields, queryFilter, stmt.IsDesc(), stmt.GetLimit())
	result, err := influx.client.QueryAPI(influx.org.Name).Query(context.Background(), queryString)
	ret := make([]map[string]interface{}, 0)
	if err == nil {
		extractDataFunc := extractDataByColumn
		if stmt.ByTime() {
			extractDataFunc = extractDataByTime
		}
		for result.Next() {
			if result.TableChanged() {
				klog.V(6).InfoS("Table changed", "table", result.TableMetadata().String())
			}
			klog.V(6).InfoS("Row", "row", result.Record().String())
			ret = extractDataFunc(ret, result.Record(), stmt)
		}
		if result.Err() != nil {
			klog.V(3).InfoS("Failed to query", "err", result.Err().Error())
		}
	} else {
		klog.V(3).InfoS("Failed to query", "err", err.Error())
	}

	return ret, nil
}

func (influx *InfluxDB) ListTables(stmt *query.Range) (interface{}, error) {
	queryTemplate := `
  from(bucket: "%s%s")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => %s)
  |> keep(columns: ["_measurement"])
  |> limit(n: 1)
`
	bucket := stmt.GetTableProperty(data.Database).(string)
	primaryKeyDialect := "true"
	for k, v := range stmt.GetKeys() {
		if stmt.TableDefinitions.IsPrimaryKey(k) {
			primaryKeyDialect = fmt.Sprintf(`r["%s"] == "%v"`, k, v)
		}
	}

	queryString := fmt.Sprintf(queryTemplate, bucket, generic.GetRPName(stmt.GetTTL()), stmt.GetStart().Format(time.RFC3339Nano), stmt.GetEnd().Format(time.RFC3339Nano), primaryKeyDialect)
	result, err := influx.client.QueryAPI(influx.org.Name).Query(context.Background(), queryString)
	ret := make([]string, 0)
	if err == nil {
		for result.Next() {
			if result.TableChanged() {
				klog.V(6).InfoS("Table changed", "table", result.TableMetadata().String())
			}
			klog.V(6).InfoS("Row", "row", result.Record().String())
			ret = append(ret, result.Record().Measurement())
		}
		if result.Err() != nil {
			klog.V(3).InfoS("Failed to query", "err", result.Err().Error())
		}
	} else {
		klog.V(3).InfoS("Failed to query", "err", err.Error())
	}
	return ret, nil
}

func (influx *InfluxDB) Insert(stmt *insert.Upsert) (interface{}, error) {
	td := stmt.TableDefinition
	bucket := stmt.GetTableProperty(data.Database).(string)
	keys := stmt.GetKeys()
	tags := make(map[string]string, len(keys))
	secondaryKeys := make(map[string]interface{})
	for k, v := range keys {
		if td.IsPrimaryKey(k) {
			if s, ok := v.(string); ok {
				tags[k] = s
			} else {
				tags[k] = fmt.Sprintf("%v", v)
			}
		} else {
			secondaryKeys[k] = v
		}
	}
	fields := rows2Fields(stmt.GetRows())
	for _, f := range fields {
		for k, v := range secondaryKeys {
			f[k] = v
		}
		// var ts time.Time
		ts, ok := f[data.TimeKey].(time.Time)
		if !ok {
			ts = time.Now()
		} else {
			delete(f, data.TimeKey)
		}
		wp := influxdb2.NewPoint(stmt.GetTableName(), tags, f, ts.UTC())
		if stmt.IsSync() {
			err := influx.client.WriteAPIBlocking(influx.org.Name, fmt.Sprintf("%s%s", bucket, generic.GetRPName(stmt.GetTTL()))).WritePoint(nil, wp)
			if err != nil {
				klog.V(3).InfoS("Failed to write record", "err", err)
			}
		} else {
			influx.client.WriteAPI(influx.org.Name, fmt.Sprintf("%s%s", bucket, generic.GetRPName(stmt.GetTTL()))).WritePoint(wp)
		}
	}
	return nil, nil
}

func (influx *InfluxDB) Update(stmt *insert.Upsert) (interface{}, error) {
	return nil, nil
}

func (influx *InfluxDB) Delete(stmt *insert.Delete) (interface{}, error) {
	bucket := stmt.GetTableProperty(data.Database).(string)

	predicate := fmt.Sprintf(`_measurement="%s"`, stmt.GetTableName())
	for k, v := range stmt.GetKeys() {
		if stmt.IsPrimaryKey(k) {
			predicate += fmt.Sprintf(` AND %s="%v"`, k, v)
		}
	}

	if err := influx.client.DeleteAPI().DeleteWithName(context.Background(), influx.org.Name, fmt.Sprintf("%s%s", bucket, generic.GetRPName(stmt.GetTTL())), stmt.GetStart(), stmt.GetEnd(), predicate); err != nil {
		klog.V(3).InfoS("Failed to delete record", "err", err)
		return nil, err
	}

	return nil, nil
}

func (influx *InfluxDB) Close() {
	influx.client.Close()
}

func (influx *InfluxDB) GetTTLSeconds(database string) int {
	if db, exist := influx.databases[database]; exist {
		if db.GetTTL() == 0 {
			return generic.MaxTTLSeconds
		}
		return db.GetTTL()
	} else {
		return 0
	}

}

func (influx *InfluxDB) GetTTLDays(database string) int {
	if db, exist := influx.databases[database]; exist {
		if db.GetTTL() == 0 {
			return generic.MaxTTLDay
		}
		return db.GetTTL() / 86400
	} else {
		return 0
	}
}

// type extractDataFunc func(ret []map[string]interface{}, record *fluxquery.FluxRecord, stmt *query.Range) []map[string]interface{}

func extractDataByTime(ret []map[string]interface{}, record *fluxquery.FluxRecord, stmt *query.Range) []map[string]interface{} {
	item := map[string]interface{}{
		data.TimeKey: record.Time(),
	}
	if stmt.ReplaceTableName() != nil {
		item[stmt.ReplaceTableName()()] = record.Measurement()
	}
	for k, v := range record.Values() {
		if stmt.GetSelects().Has(k) {
			item[k] = v
		}
	}
	return append(ret, item)
}

func extractDataByColumn(ret []map[string]interface{}, record *fluxquery.FluxRecord, _ *query.Range) []map[string]interface{} {
	column := record.Field()
	if len(ret) == 0 {
		ret = append(ret, make(map[string]interface{}))
	}
	var fields []data.ResultColumn
	if v, ok := ret[0][column]; ok {
		fields = v.([]data.ResultColumn)
	}
	fields = append(fields, data.ResultColumn{
		Value: record.Value(),
		Time:  record.Time(),
	})
	ret[0][column] = fields
	return ret
}

func rows2Fields(rows []*data.Row) []map[string]interface{} {
	fields := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		fields[i] = r.GetValues()
	}
	return fields
}
