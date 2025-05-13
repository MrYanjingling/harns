package thing

import (
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/model/runtime"
	"sort"
	"strings"
)

type lessTypeFunc func(tt1, tt2 *runtime.ThingType) bool

type typeSorter struct {
	tts       []*runtime.ThingType
	lessFuncs []lessTypeFunc
}

func ByType(less ...lessTypeFunc) *typeSorter {
	return &typeSorter{
		lessFuncs: less,
	}
}
func (ms *typeSorter) Sort(tts []*runtime.ThingType) {
	ms.tts = tts
	sort.Sort(ms)
}

func (ms *typeSorter) Len() int {
	return len(ms.tts)
}

func (ms *typeSorter) Swap(i, j int) {
	ms.tts[i], ms.tts[j] = ms.tts[j], ms.tts[i]
}

func (ms *typeSorter) Less(i, j int) bool {
	return ms.less(ms.tts[i], ms.tts[j])
}

func (ms *typeSorter) less(p, q *runtime.ThingType) bool {
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

func (ms *typeSorter) Insert(tts []*runtime.ThingType, tt *runtime.ThingType) []*runtime.ThingType {
	i := sort.Search(len(tts), func(i int) bool { return ms.less(tts[i], tt) })
	tts = append(tts, &runtime.ThingType{})
	copy(tts[i+1:], tts[i:])
	tts[i] = tt
	return tts
}

type ThingTypeFilter struct {
	Tenant       string
	Id           string
	ParentTypeId string
	Name         interface{}
}

type predicateType func(t *runtime.ThingType) bool

func parseTypeFilter(filter *ThingTypeFilter) []predicateType {
	// tenant
	predicates := []predicateType{func(tt *runtime.ThingType) bool {
		if filter.Tenant == tt.Tenant {
			return true
		}
		return false
	}}
	// id
	if len(filter.Id) > 0 {
		p := func(tt *runtime.ThingType) bool {
			if filter.Id == tt.ID {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	// parentTypeId
	if len(filter.ParentTypeId) > 0 {
		p := func(tt *runtime.ThingType) bool {
			if filter.ParentTypeId == tt.ParentTypeId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(tt *runtime.ThingType) bool {
				if name == tt.Name {
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
				p := func(tt *runtime.ThingType) bool {
					if ff.Eq == tt.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(tt *runtime.ThingType) bool {
					for _, name := range ff.In {
						if name == tt.Name {
							return true
						}
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// contains
			if len(ff.Contains) > 0 {
				p := func(tt *runtime.ThingType) bool {
					if strings.Contains(tt.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(tt *runtime.ThingType) bool {
					if strings.HasPrefix(tt.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(tt *runtime.ThingType) bool {
					if strings.HasSuffix(tt.Name, strings.TrimSpace(ff.EndsWith)) {
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

// thing filter
type lessFunc func(tt1, tt2 *runtime.Thing) bool

type sorter struct {
	ts        []*runtime.Thing
	lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
	return &sorter{
		lessFuncs: less,
	}
}
func (ms *sorter) Sort(psts []*runtime.Thing) {
	ms.ts = psts
	sort.Sort(ms)
}

func (ms *sorter) Len() int {
	return len(ms.ts)
}

func (ms *sorter) Swap(i, j int) {
	ms.ts[i], ms.ts[j] = ms.ts[j], ms.ts[i]
}

func (ms *sorter) Less(i, j int) bool {
	return ms.less(ms.ts[i], ms.ts[j])
}

func (ms *sorter) less(p, q *runtime.Thing) bool {
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

func (ms *sorter) Insert(ts []*runtime.Thing, t *runtime.Thing) []*runtime.Thing {
	i := sort.Search(len(ts), func(i int) bool { return ms.less(ts[i], t) })
	ts = append(ts, &runtime.Thing{})
	copy(ts[i+1:], ts[i:])
	ts[i] = t
	return ts
}

type thingFilter struct {
	Tenant   string
	Id       string
	TypeId   string
	ParentId string
	Name     interface{}
	HasType  string
}

type predicate func(t *runtime.Thing) bool

func parseFilter(filter *thingFilter, tm *Manager) []predicate {
	// tenant
	predicates := []predicate{func(t *runtime.Thing) bool {
		if filter.Tenant == t.Tenant {
			return true
		}
		return false
	}}
	// id
	if len(filter.Id) > 0 {
		p := func(t *runtime.Thing) bool {
			if filter.Id == t.ID {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	// typeId
	if len(filter.TypeId) > 0 {
		p := func(t *runtime.Thing) bool {
			if t.Type != nil && filter.TypeId == t.Type.ID {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	// parentId
	if len(filter.ParentId) > 0 {
		p := func(t *runtime.Thing) bool {
			if t.Parent != nil && filter.ParentId == t.Parent.ID {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	// name
	if filter.Name != nil {
		if name, ok := filter.Name.(string); ok {
			p := func(t *runtime.Thing) bool {
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
				p := func(t *runtime.Thing) bool {
					if ff.Eq == t.Name {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// in
			if len(ff.In) > 0 {
				p := func(t *runtime.Thing) bool {
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
				p := func(t *runtime.Thing) bool {
					if strings.Contains(t.Name, ff.Contains) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// startsWith
			if len(ff.StartsWith) > 0 {
				p := func(t *runtime.Thing) bool {
					if strings.HasPrefix(t.Name, strings.TrimSpace(ff.StartsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
			// endsWith
			if len(ff.EndsWith) > 0 {
				p := func(t *runtime.Thing) bool {
					if strings.HasSuffix(t.Name, strings.TrimSpace(ff.EndsWith)) {
						return true
					}
					return false
				}
				predicates = append(predicates, p)
			}
		}
	}

	// hasType
	if len(filter.HasType) > 0 {
		self, err := tm.GetThingTypeById(filter.HasType, false)
		if err != nil {
			return predicates
		}
		var ctts []*runtime.ThingType
		rtts := []*runtime.ThingType{self}

		tts, _ := getTypeChildren(filter.HasType, tm)
		rtts = append(rtts, tts...)
		ctts = append(ctts, tts...)
		for len(ctts) > 0 {
			tts = ctts
			ctts = make([]*runtime.ThingType, 0)
			for _, tt := range tts {
				tts, _ = getTypeChildren(tt.ID, tm)
				ctts = append(ctts, tts...)
			}
			rtts = append(rtts, ctts...)
		}

		p := func(t *runtime.Thing) bool {
			for _, tt := range rtts {
				if t.Type.ID == tt.ID {
					return true
				}
			}
			return false
		}
		predicates = append(predicates, p)
	}

	return predicates
}

func getTypeChildren(typeId string, tm *Manager) ([]*runtime.ThingType, error) {
	ttf := &ThingTypeFilter{
		ParentTypeId: typeId,
	}
	return tm.GetThingTypes(ttf, false)
}
