package runtime

import "math"

func (p *Property) GetMinInt32() int32 {
	if p.Min == nil {
		return math.MinInt32
	} else {
		return int32(*p.Min)
	}
}

func (p *Property) GetMaxInt32() int32 {
	if p.Max == nil {
		return math.MaxInt32
	} else {
		return int32(*p.Max)
	}
}

func (p *Property) GetMinLong() int64 {
	if p.Min == nil {
		return math.MinInt64
	} else {
		return int64(*p.Min)
	}
}

func (p *Property) GetMaxLong() int64 {
	if p.Max == nil {
		return math.MaxInt64
	} else {
		return int64(*p.Max)
	}
}

func (p *Property) GetMinFloat() float64 {
	if p.Min == nil {
		return -math.MaxFloat64
	} else {
		return *p.Min
	}
}

func (p *Property) GetMaxFloat() float64 {
	if p.Max == nil {
		return math.MaxFloat64
	} else {
		return *p.Max
	}
}
