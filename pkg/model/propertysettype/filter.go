package propertysettype

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/model/runtime"
	"sort"
	"strings"
)

type lessFunc func(pst1, pst2 *runtime.PropertySetType) bool

type pstSorter struct {
	psts      []*runtime.PropertySetType
	lessFuncs []lessFunc
}

func By(less ...lessFunc) *pstSorter {
	return &pstSorter{
		lessFuncs: less,
	}
}
func (ms *pstSorter) Sort(psts []*runtime.PropertySetType) {
	ms.psts = psts
	sort.Sort(ms)
}

func (ms *pstSorter) Len() int {
	return len(ms.psts)
}

func (ms *pstSorter) Swap(i, j int) {
	ms.psts[i], ms.psts[j] = ms.psts[j], ms.psts[i]
}

func (ms *pstSorter) Less(i, j int) bool {
	return ms.less(ms.psts[i], ms.psts[j])
}

func (ms *pstSorter) less(p, q *runtime.PropertySetType) bool {
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

func (ms *pstSorter) Insert(psts []*runtime.PropertySetType, pst *runtime.PropertySetType) []*runtime.PropertySetType {
	i := sort.Search(len(psts), func(i int) bool { return ms.less(psts[i], pst) })
	psts = append(psts, &runtime.PropertySetType{})
	copy(psts[i+1:], psts[i:])
	psts[i] = pst
	return psts
}

type filter struct {
	Tenant string
	Name   interface{}
}

type predicate func(pst *runtime.PropertySetType) bool

func parseFilter(filter *filter) []predicate {
	// tenant
	predicates := []predicate{func(pst *runtime.PropertySetType) bool {
		if filter.Tenant == pst.Tenant {
			return true
		}
		return false
	}}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(pst *runtime.PropertySetType) bool {
				if name == pst.Name {
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
				p := func(pst *runtime.PropertySetType) bool {
					if ff.Eq == pst.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(pst *runtime.PropertySetType) bool {
					for _, name := range ff.In {
						if name == pst.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(pst *runtime.PropertySetType) bool {
					if strings.Contains(pst.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(pst *runtime.PropertySetType) bool {
					if strings.HasPrefix(pst.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(pst *runtime.PropertySetType) bool {
					if strings.HasSuffix(pst.Name, strings.TrimSpace(ff.EndsWith)) {
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
