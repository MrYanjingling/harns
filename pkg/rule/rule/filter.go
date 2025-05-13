package rule

import (
	"lightiot/pkg/rule/runtime"
	"sort"
)

type lessFunc func(r1, r2 *runtime.Rule) bool

type sorter struct {
	rs        []*runtime.Rule
	lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
	return &sorter{
		lessFuncs: less,
	}
}
func (s *sorter) Sort(rs []*runtime.Rule) {
	s.rs = rs
	sort.Sort(s)
}

func (s *sorter) Len() int {
	return len(s.rs)
}

func (s *sorter) Swap(i, j int) {
	s.rs[i], s.rs[j] = s.rs[j], s.rs[i]
}

func (s *sorter) Less(i, j int) bool {
	return s.less(s.rs[i], s.rs[j])
}

func (s *sorter) less(p, q *runtime.Rule) bool {
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(s.lessFuncs)-1; k++ {
		less := s.lessFuncs[k]
		switch {
		case less(p, q):
			return true
		case less(q, p):
			return false
		}
	}
	return s.lessFuncs[k](p, q)
}

func (s *sorter) Insert(rs []*runtime.Rule, r *runtime.Rule) []*runtime.Rule {
	i := sort.Search(len(rs), func(i int) bool { return s.less(rs[i], r) })
	rs = append(rs, &runtime.Rule{})
	copy(rs[i+1:], rs[i:])
	rs[i] = r
	return rs
}

type filter struct {
	Tenant  string
	ThingId string
}

type predicate func(r *runtime.Rule) bool

func parseFilter(filter *filter) []predicate {
	// tenant
	predicates := []predicate{func(r *runtime.Rule) bool {
		if filter.Tenant == r.Tenant {
			return true
		}
		return false
	}}

	// thingId
	if len(filter.ThingId) > 0 {
		p := func(r *runtime.Rule) bool {
			if filter.ThingId == r.ThingId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	return predicates
}
