package event

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/event/runtime"
	"lightiot/pkg/generic"
	"sort"
	"strings"
)

type lessTypeFunc func(tt1, tt2 *runtime.EventType) bool

type typeSorter struct {
	ets       []*runtime.EventType
	lessFuncs []lessTypeFunc
}

func ByType(less ...lessTypeFunc) *typeSorter {
	return &typeSorter{
		lessFuncs: less,
	}
}
func (ms *typeSorter) Sort(ets []*runtime.EventType) {
	ms.ets = ets
	sort.Sort(ms)
}

func (ms *typeSorter) Len() int {
	return len(ms.ets)
}

func (ms *typeSorter) Swap(i, j int) {
	ms.ets[i], ms.ets[j] = ms.ets[j], ms.ets[i]
}

func (ms *typeSorter) Less(i, j int) bool {
	return ms.less(ms.ets[i], ms.ets[j])
}

func (ms *typeSorter) less(p, q *runtime.EventType) bool {
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.lessFuncs)-1; k++ {
		less := ms.lessFuncs[k]
		switch {
		case less(p, q):
			return true
		case less(q, p):
			return false
		}
	}
	return ms.lessFuncs[k](p, q)
}

func (ms *typeSorter) Insert(ets []*runtime.EventType, et *runtime.EventType) []*runtime.EventType {
	i := sort.Search(len(ets), func(i int) bool { return ms.less(ets[i], et) })
	ets = append(ets, &runtime.EventType{})
	copy(ets[i+1:], ets[i:])
	ets[i] = et
	return ets
}

type TypeFilter struct {
	Id       string
	Name     interface{}
	parentId string
}

type predicateType func(t *runtime.EventType) bool

func parseTypeFilter(filter *TypeFilter) []predicateType {
	var predicates []predicateType
	// id
	if len(filter.Id) > 0 {
		p := func(et *runtime.EventType) bool {
			if filter.Id == et.ID {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(et *runtime.EventType) bool {
				if name == et.Name {
					return true
				}
				return false
			}
			predicates = append(predicates, p)
		} else {
			var ff generic.NameFilterFunc
			if err := mapstructure.Decode(filter.Name, &ff); err != nil {
				klog.V(4).InfoS("Failed to decode filter", "err", err)
			}
			// eq
			if len(ff.Eq) > 0 {
				p := func(et *runtime.EventType) bool {
					if name == et.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(et *runtime.EventType) bool {
					for _, name := range ff.In {
						if name == et.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(et *runtime.EventType) bool {
					if strings.Contains(et.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(et *runtime.EventType) bool {
					if strings.HasPrefix(et.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(et *runtime.EventType) bool {
					if strings.HasSuffix(et.Name, strings.TrimSpace(ff.EndsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
		}
	}

	if len(filter.parentId) > 0 {
		p := func(et *runtime.EventType) bool {
			if et.ParentId != nil && filter.parentId == *et.ParentId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	return predicates
}

// Event
