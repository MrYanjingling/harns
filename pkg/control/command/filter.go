package command

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/generic"
	"sort"
	"strings"
)

type lessTypeFunc func(ct1, ct2 *runtime.CommandType) bool

type typeSorter struct {
	cts       []*runtime.CommandType
	lessFuncs []lessTypeFunc
}

func ByType(less ...lessTypeFunc) *typeSorter {
	return &typeSorter{
		lessFuncs: less,
	}
}

func (ts *typeSorter) Sort(cts []*runtime.CommandType) {
	ts.cts = cts
	sort.Sort(ts)
}

func (ts *typeSorter) Len() int {
	return len(ts.cts)
}

func (ts *typeSorter) Swap(i, j int) {
	ts.cts[i], ts.cts[j] = ts.cts[j], ts.cts[i]
}

func (ts *typeSorter) Less(i, j int) bool {
	return ts.less(ts.cts[i], ts.cts[j])
}

func (ts *typeSorter) less(p, q *runtime.CommandType) bool {
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ts.lessFuncs)-1; k++ {
		less := ts.lessFuncs[k]
		switch {
		case less(p, q):
			return true
		case less(q, p):
			return false
		}
	}
	return ts.lessFuncs[k](p, q)
}

func (ts *typeSorter) Insert(cts []*runtime.CommandType, ct *runtime.CommandType) []*runtime.CommandType {
	i := sort.Search(len(cts), func(i int) bool { return ts.less(cts[i], ct) })
	cts = append(cts, &runtime.CommandType{})
	copy(cts[i+1:], cts[i:])
	cts[i] = ct
	return cts
}

type typeFilter struct {
	Id          string
	Tenant      string
	Name        interface{}
	ThingTypeId string
}

type predicateType func(t *runtime.CommandType) bool

func parseTypeFilter(filter typeFilter) []predicateType {
	var predicates []predicateType
	//tenant
	p := func(ct *runtime.CommandType) bool {
		if filter.Tenant == ct.Tenant {
			return true
		}
		return false
	}
	predicates = append(predicates, p)

	// id
	if len(filter.Id) > 0 {
		p := func(ct *runtime.CommandType) bool {
			if filter.Id == ct.TypeId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(ct *runtime.CommandType) bool {
				if name == ct.Name {
					return true
				}
				return false
			}
			predicates = append(predicates, p)
		} else {
			var ff generic.NameFilterFunc
			if err := mapstructure.Decode(filter.Name, &ff); err != nil {
				klog.V(3).InfoS("Failed to parse filter", "err", err)
			}
			// eq
			if len(ff.Eq) > 0 {
				p := func(ct *runtime.CommandType) bool {
					if name == ct.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(ct *runtime.CommandType) bool {
					for _, name := range ff.In {
						if name == ct.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(ct *runtime.CommandType) bool {
					if strings.Contains(ct.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(ct *runtime.CommandType) bool {
					if strings.HasPrefix(ct.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(ct *runtime.CommandType) bool {
					if strings.HasSuffix(ct.Name, strings.TrimSpace(ff.EndsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
		}
	}
	// thingTypeId
	if len(filter.ThingTypeId) > 0 {
		p := func(ct *runtime.CommandType) bool {
			if filter.ThingTypeId == ct.ThingTypeId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	return predicates
}
