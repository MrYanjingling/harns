package storage

import (
	"context"
	"fmt"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/consumer/job"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/propertysettype"
	"lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	v1 "lightiot/pkg/model/v1"
	"path"
	"strconv"
	"time"
)

type RuleEvalFunc func(key string, rawData RawData)

type Store struct {
	tm   *thing.Manager
	pstm *propertysettype.Manager
	jm   *job.Manager

	client       influxdb2.Client
	syncWriter   api.WriteAPIBlocking
	asyncWriter  api.WriteAPI
	reader       api.QueryAPI
	delete       api.DeleteAPI
	ruleEvalFunc RuleEvalFunc
}

type Option func(*Store)

func WithJobManager(jm *job.Manager) Option {
	return func(s *Store) {
		s.jm = jm
	}
}

func WithRuleEvalFunc(f RuleEvalFunc) Option {
	return func(s *Store) {
		s.ruleEvalFunc = f
	}
}

func NewStore(stopCh <-chan struct{}, influxdbUrl, influxdbToken string, opts ...Option) *Store {
	client := influxdb2.NewClientWithOptions(influxdbUrl,
		influxdbToken,
		influxdb2.DefaultOptions().
			SetPrecision(time.Millisecond))
	s := &Store{
		client:      client,
		syncWriter:  client.WriteAPIBlocking("main", timeSeriesRawDataBucket),
		asyncWriter: client.WriteAPI("main", timeSeriesRawDataBucket),
		delete:      client.DeleteAPI(),
		reader:      client.QueryAPI("main"),
	}
	for _, opt := range opts {
		opt(s)
	}
	ttCh := make(chan gruntime.Object)
	tCh := make(chan gruntime.Object)
	pstCh := make(chan gruntime.Object)
	pstm := propertysettype.NewManager(stopCh, pstCh)

	s.tm = thing.NewManager(stopCh, thing.WithThingTypeChan(ttCh), thing.WithThingChan(tCh), thing.WithPropertySetTypeManager(pstm), thing.WithJobManager(s.jm))
	s.pstm = pstm
	return s
}

func (s *Store) Init() {
	s.pstm.InitWatch()
	s.tm.InitWatch()
}

func (s *Store) Close() {
	// it shall flush all async point
	s.client.Close()
}

func (s *Store) SaveOrUpdateTimeSeries(thingId, psName string, data []map[string]interface{}, sync bool) error {
	thing, err := s.tm.GetThingById(thingId)
	if err != nil {
		klog.V(3).InfoS("Thing not found", "id", thingId)
		return response.ErrThingNotFound(thingId)
	}
	ps, ok := thing.PropertySetByName[psName]
	if !ok {
		klog.V(3).InfoS("PropertySet not found", "propertySet", psName, "thingId", thingId)
		return response.ErrPropertySetNotFound(psName)
	}
	propertiesByName := ps.PropertySetType.PropertyByName
	rawData := make(RawData)
	for _, item := range data {
		isLegal := false
		rawTime := time.Now()
		rawItem := make(map[string]interface{})
		for k, v := range item {
			if p, ok := propertiesByName[k]; ok {
				rawItem[k] = convertDataAccordingToType(p, v)
			} else if k == TimeKey {
				isLegal = true
				rawTime, err = time.Parse(time.RFC3339Nano, v.(string))
				if err != nil {
					isLegal = false
					klog.V(3).InfoS("Failed to parse time", "err", err)
				}
			} else {
				klog.V(3).InfoS("Property not found", "property", k, "propertySet", psName, "thingId", thingId)
			}
		}
		if isLegal {
			rawTime = rawTime.UTC()
			rawData[rawTime] = rawItem
		} else {
			klog.V(3).InfoS("Invalid time series", "item", item)
		}
	}

	measurement := ps.PropertySetType.ID
	for k, v := range rawData {
		wp := influxdb2.NewPoint(measurement, map[string]string{"ps": psName, "ti": thingId}, v, k)
		if sync {
			err = s.syncWriter.WritePoint(nil, wp)
			if err != nil {
				klog.V(3).InfoS("Failed to write time series", "err", err)
			}
		} else {
			s.asyncWriter.WritePoint(wp)
		}
	}

	s.ruleEvalFunc(path.Join(thingId, psName), rawData)

	if s.jm != nil {
		go func(key string, rawData RawData, loc *time.Location) {
			now := time.Now()
			for k := range rawData {
				_ = s.jm.Rollup1MinuteJob(key, now, k, loc)
				_ = s.jm.Rollup1HourJob(key, now, k, loc)
				_ = s.jm.Rollup1DayJob(key, now, k, loc)
			}
		}(path.Join(thingId, psName), rawData, (*time.Location)(thing.TimeZone))
	}

	return nil
}

func (s *Store) GetTimeSeries(thingId, psName string, from, to time.Time, selects []string, desc, latest bool, limit int) ([]map[string]interface{}, error) {
	thing, err := s.tm.GetThingById(thingId)
	if err != nil {
		klog.V(3).InfoS("Thing not found", "id", thingId)
		return nil, response.ErrThingNotFound(thingId)
	}
	ps, ok := thing.PropertySetByName[psName]
	if !ok {
		klog.V(3).InfoS("PropertySet not found", "propertySet", psName, "thingId", thingId)
		return nil, response.ErrPropertySetNotFound(psName)
	}

	propertiesByName := ps.PropertySetType.PropertyByName
	legalProperties := sets.String{}
	if len(selects) > 0 {
		for _, p := range selects {
			if _, ok := propertiesByName[p]; ok {
				legalProperties.Insert(p)
			} else {
				klog.V(3).InfoS("Property not found", "property", p, "propertySet", psName, "thingId", thingId)
			}
		}
	} else {
		for k := range propertiesByName {
			legalProperties.Insert(k)
		}
	}

	if legalProperties.Len() == 0 {
		return nil, response.ErrAllSelectInvalid
	}

	queryFields := ""
	for i, p := range legalProperties.UnsortedList() {
		if i != 0 {
			queryFields += " or"
		}
		queryFields += fmt.Sprintf(` r._field == "%s"`, p)
	}

	var queryString string
	if !latest {
		queryTemplate := `from(bucket:"data-raw")
|> range(start: %s, stop: %s)
|> filter(fn: (r) => r._measurement == "%s" and r.ti=="%s" and r.ps == "%s")
|> filter(fn: (r) => %s)
|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
|> sort(columns:["_time"], desc: %t)
|> limit(n:%d)`

		queryString = fmt.Sprintf(queryTemplate, from.Format(time.RFC3339Nano), to.Format(time.RFC3339Nano), ps.PropertySetType.ID, thingId, psName, queryFields, desc, limit)

		result, err := s.reader.Query(context.Background(), queryString)
		data := make([]map[string]interface{}, 0)
		if err == nil {
			for result.Next() {
				if result.TableChanged() {
					klog.V(6).InfoS("Table changed", "table", result.TableMetadata().String())
				}
				klog.V(6).InfoS("Row", "row", result.Record().String())

				item := map[string]interface{}{
					TimeKey: result.Record().Time(),
				}
				for k, v := range result.Record().Values() {
					if legalProperties.Has(k) {
						item[k] = v
					}
				}
				data = append(data, item)
			}
			if result.Err() != nil {
				klog.V(3).InfoS("Failed to query", "err", result.Err().Error())
			}
		} else {
			klog.V(3).InfoS("Failed to query", "err", err.Error())
		}

		return data, nil
	} else {
		queryTemplate := `from(bucket:"data-raw")
|> range(start: -7d)
|> filter(fn: (r) => r._measurement == "%s" and r.ti=="%s" and r.ps == "%s")
|> filter(fn: (r) => %s)
|> last()`

		queryString = fmt.Sprintf(queryTemplate, ps.PropertySetType.ID, thingId, psName, queryFields)

		result, err := s.reader.Query(context.Background(), queryString)
		data := make(RawData, 0)
		if err == nil {
			for result.Next() {
				if result.TableChanged() {
					klog.V(6).InfoS("Table changed", "table", result.TableMetadata().String())
				}
				klog.V(6).InfoS("Row", "row", result.Record().String())

				t := result.Record().Time()
				if _, ok := data[t]; !ok {
					data[t] = make(map[string]interface{}, 0)
					data[t][TimeKey] = t
				}
				data[t][result.Record().Field()] = result.Record().Value()
			}
			if result.Err() != nil {
				klog.V(3).InfoS("Failed to query", "err", result.Err().Error())
			}
		} else {
			klog.V(3).InfoS("Failed to query", "err", err.Error())
		}

		res := make([]map[string]interface{}, 0)

		for _, v := range data {
			res = append(res, v)
		}
		return res, nil
	}
}

func (s *Store) SaveOrUpdateVirtualParameter(thingId, psName string, property *runtime.Property, t time.Time, v float64) error {
	value := convertDataAccordingToType(property, v)
	wp := influxdb2.NewPoint(thingId, map[string]string{"ps": psName}, map[string]interface{}{property.Name: value}, t.UTC())
	s.asyncWriter.WritePoint(wp)
	return nil
}

func (s *Store) DeleteTimeSeries(thingId, psName string, start, end time.Time) error {
	t, err := s.tm.GetThingById(thingId)
	if err != nil {
		klog.V(3).InfoS("Thing not found", "id", thingId)
		return response.ErrThingNotFound(thingId)
	}
	ps, ok := t.PropertySetByName[psName]
	if !ok {
		klog.V(3).InfoS("PropertySet not found", "propertySet", psName, "thingId", thingId)
		return response.ErrPropertySetNotFound(psName)
	}

	go func() {
		// todo should delete by Batches
		predicate := fmt.Sprintf(`_measurement="%s" AND ps="%s" AND ti="%s"`, ps.PropertySetType.ID, psName, thingId)
		if err := s.delete.DeleteWithName(context.Background(), "main", timeSeriesRawDataBucket, start, end, predicate); err != nil {
			klog.V(3).InfoS("Failed to delete time series", "err", err)
		}
	}()
	return nil
}

var convertFuncs = map[v1.Datatype]func(*runtime.Property, interface{}) interface{}{
	v1.DataTypeString: func(p *runtime.Property, v interface{}) interface{} {
		s, ok := v.(string)
		if !ok {
			s = fmt.Sprintf("%v", v)
		}
		rs := []rune(s)
		if len(rs) > p.Length {
			return string(rs[0:p.Length])
		} else {
			return s
		}
	},
	v1.DataTypeInt: func(p *runtime.Property, v interface{}) interface{} {
		switch i := v.(type) {
		case float64:
			return int(i)
		case string:
			if value, err := strconv.Atoi(i); err != nil {
				klog.V(3).InfoS("Failed parsing to int", "value", v, "err", err)
				return 0
			} else {
				return value
			}
		default:
			klog.V(3).InfoS("Failed parsing to int", "value", v)
			return 0
		}
	},
	v1.DataTypeLong: func(p *runtime.Property, v interface{}) interface{} {
		switch l := v.(type) {
		case float64:
			return int64(l)
		case string:
			if value, err := strconv.ParseInt(l, 10, 64); err != nil {
				klog.V(3).InfoS("Failed parsing to long", "value", v, "err", err)
				return int64(0)
			} else {
				return value
			}
		default:
			klog.V(3).InfoS("Failed parsing to long", "value", v)
			return int64(0)
		}
	},
	v1.DataTypeDouble: func(p *runtime.Property, v interface{}) interface{} {
		switch d := v.(type) {
		case float64:
			return d
		case string:
			if value, err := strconv.ParseFloat(d, 64); err != nil {
				klog.V(3).InfoS("Failed parsing to double", "value", v, "err", err)
				return float64(0)
			} else {
				return value
			}
		default:
			klog.V(3).InfoS("Failed parsing to double", "value", v)
			return float64(0)
		}
	},
	v1.DataTypeBoolean: func(p *runtime.Property, v interface{}) interface{} {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			if value, err := strconv.ParseBool(b); err != nil {
				klog.V(3).InfoS("Failed parsing to bool", "value", v, "err", err)
				return false
			} else {
				return value
			}
		case float64:
			if int(b) == 0 {
				return false
			} else {
				return true
			}
		default:
			klog.V(3).InfoS("Failed parsing to bool", "value", v)
			return false
		}
	},
}

func convertDataAccordingToType(p *runtime.Property, v interface{}) interface{} {
	if f, ok := convertFuncs[p.DataType]; ok {
		return f(p, v)
	} else {
		klog.V(3).InfoS("Unsupported datatype", "datatype", p.DataType)
		return nil
	}
}
