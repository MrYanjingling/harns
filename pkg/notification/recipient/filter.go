package recipient

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/notification/runtime"
	"sort"
	"strings"
)

type lessFunc func(r1, r2 *runtime.Recipient) bool

type sorter struct {
	rs        []*runtime.Recipient
	lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
	return &sorter{
		lessFuncs: less,
	}
}
func (s *sorter) Sort(rs []*runtime.Recipient) {
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

func (s *sorter) less(p, q *runtime.Recipient) bool {
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

func (s *sorter) Insert(rs []*runtime.Recipient, recipient *runtime.Recipient) []*runtime.Recipient {
	i := sort.Search(len(rs), func(i int) bool { return s.less(rs[i], recipient) })
	rs = append(rs, &runtime.Recipient{})
	copy(rs[i+1:], rs[i:])
	rs[i] = recipient
	return rs
}

type Filter struct {
	Tenant string
	Name   interface{}
}

type predicate func(i *runtime.Recipient) bool

func parseFilter(filter Filter) []predicate {
	// tenant
	predicates := []predicate{func(r *runtime.Recipient) bool {
		if filter.Tenant == r.Tenant {
			return true
		}
		return false
	}}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(r *runtime.Recipient) bool {
				if name == r.Name {
					return true
				}
				return false
			}
			predicates = append(predicates, p)
		} else {
			var ff generic.NameFilterFunc
			if err := mapstructure.Decode(filter.Name, &ff); err != nil {
				klog.V(3).InfoS("Failed to parse filter.name", "err", err)
			}
			// eq
			if len(ff.Eq) > 0 {
				p := func(r *runtime.Recipient) bool {
					if ff.Eq == r.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(r *runtime.Recipient) bool {
					for _, name := range ff.In {
						if name == r.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(r *runtime.Recipient) bool {
					if strings.Contains(r.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(r *runtime.Recipient) bool {
					if strings.HasPrefix(r.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(r *runtime.Recipient) bool {
					if strings.HasSuffix(r.Name, strings.TrimSpace(ff.EndsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
		}
	}
	return predicates
}
