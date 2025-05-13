package runtime

import "math"

func (o *Option) GetMinInt() int64 {
	if o.Min == nil {
		return math.MinInt64
	} else {
		return int64(*o.Min)
	}
}

func (o *Option) GetMaxInt() int64 {
	if o.Max == nil {
		return math.MaxInt64
	} else {
		return int64(*o.Max)
	}
}

func (o *Option) GetMinFloat() float64 {
	if o.Min == nil {
		return -math.MaxFloat64
	} else {
		return *o.Min
	}
}

func (o *Option) GetMaxFloat() float64 {
	if o.Max == nil {
		return math.MaxFloat64
	} else {
		return *o.Max
	}
}

