package data

import "time"

type Range struct {
	startIncl time.Time
	endExcl   time.Time
}

func NewRange(start, end time.Time) *Range {
	return &Range{start, end}
}

func (r *Range) GetStart() time.Time {
	return r.startIncl
}

func (r *Range) GetEnd() time.Time {
	return r.endExcl
}