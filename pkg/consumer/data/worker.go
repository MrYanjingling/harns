package data

import (
	"context"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/runtime"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/model/propertysettype"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	"time"
)

type Worker struct {
	pstm        *propertysettype.Manager
	tm          *thing.Manager
	asyncWriter api.WriteAPI
	reader      api.QueryAPI
	delete      api.DeleteAPI
	config      *Config
	logStore    logstorage.Interface

	cache *lru.Cache
}

func NewWorker(stopCh <-chan struct{}, config *Config, logStore logstorage.Interface) *Worker {
	pstCh := make(chan runtime.Object)
	ttCh := make(chan runtime.Object)
	tCh := make(chan runtime.Object)

	cache, err := lru.New(cacheSize)
	if err != nil {
		klog.InfoS("Failed to create cache", "err", err)
	}

	w := &Worker{
		asyncWriter: config.Client.WriteAPI("main", timeSeriesRawDataBucket),
		delete:      config.Client.DeleteAPI(),
		reader:      config.Client.QueryAPI("main"),
		config:      config,
		logStore:    logStore,
		cache:       cache,
	}

	w.pstm = propertysettype.NewManager(stopCh, pstCh,
		propertysettype.WithUpdateCacheFunc(func(pst *model.PropertySetType, et model.EventType) { w.updateCache(pst, et) }),
	)

	w.tm = thing.NewManager(stopCh,
		thing.WithThingTypeChan(ttCh),
		thing.WithThingChan(tCh),
		thing.WithPropertySetTypeManager(w.pstm),
		thing.WithUpdateCacheFunc(func(pst *model.PropertySetType, et model.EventType) { w.updateCache(pst, et) }),
	)

	return w
}

func (w *Worker) Init() {
	// TODO: should only cache things which enable rollup, numeric and bool properties
	w.pstm.InitWatch()
	w.tm.InitWatch()
}

// getRawTimeSeries is different with iot-data-query. Here it groups record by property. It groups record by _time in iot-data-query.
func (w *Worker) getRawTimeSeries(thingId, psName, psId string, start, end time.Time, properties []string) (data map[string][]field) {
	queryFields := ""
	for i, p := range properties {
		if i != 0 {
			queryFields += " or"
		}
		queryFields += fmt.Sprintf(` r._field == "%s"`, p)
	}

	var queryString string
	queryTemplate := `from(bucket:"data-raw")
|> range(start: %s, stop: %s)
|> filter(fn: (r) => r._measurement == "%s" and r.ps == "%s" and r.ti == "%s")
|> filter(fn: (r) => %s)`

	queryString = fmt.Sprintf(queryTemplate, start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano), psId, psName, thingId, queryFields)

	result, err := w.reader.Query(context.Background(), queryString)
	if err == nil {
		data = make(map[string][]field, len(properties))
		for _, p := range properties {
			data[p] = make([]field, 0)
		}

		for result.Next() {
			if result.TableChanged() {
				klog.V(6).InfoS("Table changed", "table", result.TableMetadata().String())
			}
			klog.V(6).InfoS("Row", "row", result.Record().String())

			property := result.Record().Field()
			data[property] = append(data[property], field{
				value: result.Record().Value(),
				time:  result.Record().Time(),
			})
		}
		if result.Err() != nil {
			klog.V(3).InfoS("Failed to query", "err", result.Err().Error())
		}
	} else {
		klog.V(3).InfoS("Failed to query", "err", err.Error())
	}

	return data
}
