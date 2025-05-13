package template

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/notification/runtime"
	"sort"
	"strings"
)

type lessFunc func(t1, t2 *runtime.Template) bool

type sorter struct {
	ts        []*runtime.Template
	lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
	return &sorter{
		lessFuncs: less,
	}
}
func (s *sorter) Sort(ts []*runtime.Template) {
	s.ts = ts
	sort.Sort(s)
}

func (s *sorter) Len() int {
	return len(s.ts)
}

func (s *sorter) Swap(i, j int) {
	s.ts[i], s.ts[j] = s.ts[j], s.ts[i]
}

func (s *sorter) Less(i, j int) bool {
	return s.less(s.ts[i], s.ts[j])
}

func (s *sorter) less(p, q *runtime.Template) bool {
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

func (s *sorter) Insert(ts []*runtime.Template, tmpl *runtime.Template) []*runtime.Template {
	i := sort.Search(len(ts), func(i int) bool { return s.less(ts[i], tmpl) })
	ts = append(ts, &runtime.Template{})
	copy(ts[i+1:], ts[i:])
	ts[i] = tmpl
	return ts
}

type Filter struct {
	Tenant string
	Name   interface{}
}

type predicate func(t *runtime.Template) bool

func parseFilter(filter Filter) []predicate {
	// tenant
	predicates := []predicate{func(t *runtime.Template) bool {
		if filter.Tenant == t.Tenant {
			return true
		}
		return false
	}}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(t *runtime.Template) bool {
				if name == t.Name {
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
				p := func(t *runtime.Template) bool {
					if ff.Eq == t.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(t *runtime.Template) bool {
					for _, name := range ff.In {
						if name == t.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(t *runtime.Template) bool {
					if strings.Contains(t.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(t *runtime.Template) bool {
					if strings.HasPrefix(t.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(t *runtime.Template) bool {
					if strings.HasSuffix(t.Name, strings.TrimSpace(ff.EndsWith)) {
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
