package realtimecomputation

import (
	"context"
	"k8s.io/klog/v2"
	iotts "lightiot/pkg/data/storage"
	"lightiot/pkg/promql/labels"
	"lightiot/pkg/promql/storage"
	"lightiot/pkg/tsdb/tsdbutil"
	"math"
	"strconv"
	"time"
)

type stream struct {
	rawData  iotts.RawData
	operands []string
}

// Querier implements storage.Queryable interface
func (s *stream) Querier(ctx context.Context, mint, maxt int64) (storage.Querier, error) {
	return newStreamQuerier(s.rawData, s.operands), nil
}

func newStreamQuerier(rawData iotts.RawData, operands []string) storage.Querier {
	return &streamQuerier{rawData, operands}
}

type streamQuerier struct {
	rawData  iotts.RawData
	operands []string
}

// Select returns a set of series that matches the given label matchers.
func (q *streamQuerier) Select(sortSeries bool, hints *storage.SelectHints, matchers ...*labels.Matcher) storage.SeriesSet {
	series := []storage.Series{}
	for _, m := range matchers {
		i := 0
		operand := m.Value[1:]
		index, err := strconv.Atoi(operand)
		if err != nil {
			klog.V(3).InfoS("Failed to convert to int", "source", operand, "err", err)
		}
		for k, v := range q.rawData {
			if value, ok := v[q.operands[index]]; ok {
				// i => index, o => operand
				l := labels.Labels{{Name: "i", Value: strconv.Itoa(i)}, {Name: "o", Value: m.Value}}
				series = append(series, storage.NewListSeries(l, []tsdbutil.Sample{sample{t: k.UnixNano() / int64(time.Millisecond), v: getFloat(value)}}))
				i++
			}
		}
	}
	return newStreamSeriesSet(series...)
}

// LabelValues returns all potential values for a label name.
// If matchers are specified the returned result set is reduced
// to label values of metrics matching the matchers.
func (q *streamQuerier) LabelValues(name string, matchers ...*labels.Matcher) ([]string, storage.Warnings, error) {
	return nil, storage.Warnings{}, nil
}

// LabelNames returns all the unique label names present in all queriers in sorted order.
func (q *streamQuerier) LabelNames() ([]string, storage.Warnings, error) {
	return nil, storage.Warnings{}, nil
}

// Close releases the resources of the generic querier.
func (q *streamQuerier) Close() error {
	return nil
}

type streamSeriesSet struct {
	idx    int
	series []storage.Series
}

func newStreamSeriesSet(series ...storage.Series) storage.SeriesSet {
	return &streamSeriesSet{
		idx:    -1,
		series: series,
	}
}

func (s *streamSeriesSet) Next() bool {
	s.idx++
	return s.idx < len(s.series)
}

// At returns full series. Returned series should be iterable even after Next is called.
func (s *streamSeriesSet) At() storage.Series {
	return s.series[s.idx]
}

// The error that iteration as failed with.
// When an error occurs, set cannot continue to iterate.
func (s *streamSeriesSet) Err() error {
	return nil
}

// A collection of warnings for the whole set.
// Warnings could be return even iteration has not failed with error.
func (s *streamSeriesSet) Warnings() storage.Warnings {
	return nil
}

type sample struct {
	t int64
	v float64
}

func (s sample) T() int64 {
	return s.t
}

func (s sample) V() float64 {
	return s.v
}

// https://stackoverflow.com/questions/20767724/converting-unknown-interface-to-float64-in-golang
func getFloat(unk interface{}) float64 {
	switch i := unk.(type) {
	case float64:
		return i
	case float32:
		return float64(i)
	case int64:
		return float64(i)
	case int32:
		return float64(i)
	case int:
		return float64(i)
	case uint64:
		return float64(i)
	case uint32:
		return float64(i)
	case uint:
		return float64(i)
	case int8:
		return float64(i)
	case uint8:
		return float64(i)
	case bool:
		if i {
			return float64(1)
		}
		return float64(0)
	default:
		klog.V(2).InfoS("Non-numeric type could not be converted to float")
		return math.NaN()
	}
}
