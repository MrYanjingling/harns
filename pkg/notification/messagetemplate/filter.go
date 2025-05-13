package messagetemplate

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/notification/runtime"
	"sort"
	"strings"
)

type lessFunc func(mt1, mt2 *runtime.MessageTemplate) bool

type sorter struct {
	mts       []*runtime.MessageTemplate
	lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
	return &sorter{
		lessFuncs: less,
	}
}
func (s *sorter) Sort(mts []*runtime.MessageTemplate) {
	s.mts = mts
	sort.Sort(s)
}

func (s *sorter) Len() int {
	return len(s.mts)
}

func (s *sorter) Swap(i, j int) {
	s.mts[i], s.mts[j] = s.mts[j], s.mts[i]
}

func (s *sorter) Less(i, j int) bool {
	return s.less(s.mts[i], s.mts[j])
}

func (s *sorter) less(p, q *runtime.MessageTemplate) bool {
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

func (s *sorter) Insert(mts []*runtime.MessageTemplate, msgTmpl *runtime.MessageTemplate) []*runtime.MessageTemplate {
	i := sort.Search(len(mts), func(i int) bool { return s.less(mts[i], msgTmpl) })
	mts = append(mts, &runtime.MessageTemplate{})
	copy(mts[i+1:], mts[i:])
	mts[i] = msgTmpl
	return mts
}

type Filter struct {
	Tenant string
	Name   interface{}
}

type predicate func(i *runtime.MessageTemplate) bool

func parseFilter(filter Filter) []predicate {
	// tenant
	predicates := []predicate{func(mt *runtime.MessageTemplate) bool {
		if filter.Tenant == mt.Tenant {
			return true
		}
		return false
	}}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(mt *runtime.MessageTemplate) bool {
				if name == mt.Name {
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
				p := func(mt *runtime.MessageTemplate) bool {
					if ff.Eq == mt.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(mt *runtime.MessageTemplate) bool {
					for _, name := range ff.In {
						if name == mt.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(mt *runtime.MessageTemplate) bool {
					if strings.Contains(mt.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(mt *runtime.MessageTemplate) bool {
					if strings.HasPrefix(mt.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(mt *runtime.MessageTemplate) bool {
					if strings.HasSuffix(mt.Name, strings.TrimSpace(ff.EndsWith)) {
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
