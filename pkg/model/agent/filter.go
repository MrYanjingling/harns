package agent

import (
    "lightiot/pkg/model/runtime"
    "sort"
)

type lessTypeFunc func(at1, at2 *runtime.AgentType) bool

type typeSorter struct {
    ats       []*runtime.AgentType
    lessFuncs []lessTypeFunc
}

func ByType(less ...lessTypeFunc) *typeSorter {
    return &typeSorter{
        lessFuncs: less,
    }
}
func (ms *typeSorter) Sort(tts []*runtime.AgentType) {
    ms.ats = tts
    sort.Sort(ms)
}

func (ms *typeSorter) Len() int {
    return len(ms.ats)
}

func (ms *typeSorter) Swap(i, j int) {
    ms.ats[i], ms.ats[j] = ms.ats[j], ms.ats[i]
}

func (ms *typeSorter) Less(i, j int) bool {
    return ms.less(ms.ats[i], ms.ats[j])
}

func (ms *typeSorter) less(p, q *runtime.AgentType) bool {
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

func (ms *typeSorter) Insert(ats []*runtime.AgentType, at *runtime.AgentType) []*runtime.AgentType {
    i := sort.Search(len(ats), func(i int) bool { return ms.less(ats[i], at) })
    ats = append(ats, &runtime.AgentType{})
    copy(ats[i+1:], ats[i:])
    ats[i] = at
    return ats
}

// Agent filter
type lessFunc func(tt1, tt2 *runtime.Agent) bool

type sorter struct {
    as        []*runtime.Agent
    lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
    return &sorter{
        lessFuncs: less,
    }
}
func (ms *sorter) Sort(as []*runtime.Agent) {
    ms.as = as
    sort.Sort(ms)
}

func (ms *sorter) Len() int {
    return len(ms.as)
}

func (ms *sorter) Swap(i, j int) {
    ms.as[i], ms.as[j] = ms.as[j], ms.as[i]
}

func (ms *sorter) Less(i, j int) bool {
    return ms.less(ms.as[i], ms.as[j])
}

func (ms *sorter) less(p, q *runtime.Agent) bool {
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

func (ms *sorter) Insert(as []*runtime.Agent, a *runtime.Agent) []*runtime.Agent {
    i := sort.Search(len(as), func(i int) bool { return ms.less(as[i], a) })
    as = append(as, &runtime.Agent{})
    copy(as[i+1:], as[i:])
    as[i] = a
    return as
}
