package installation

import (
    v1 "lightiot/pkg/installation/v1"
    "sort"
)

type lessFunc func(i1, i2 *Installation) bool

type sorter struct {
    is        []*Installation
    lessFuncs []lessFunc
}

func By(less ...lessFunc) *sorter {
    return &sorter{
        lessFuncs: less,
    }
}
func (s *sorter) Sort(is []*Installation) {
    s.is = is
    sort.Sort(s)
}

func (s *sorter) Len() int {
    return len(s.is)
}

func (s *sorter) Swap(i, j int) {
    s.is[i], s.is[j] = s.is[j], s.is[i]
}

func (s *sorter) Less(i, j int) bool {
    return s.less(s.is[i], s.is[j])
}

func (s *sorter) less(p, q *Installation) bool {
    // Try all but the last comparison.
    var k int
    for k = 0; k < len(s.lessFuncs) - 1; k++ {
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

func (s *sorter) Insert(is []*Installation, install *Installation) []*Installation {
    i := sort.Search(len(is), func(i int) bool { return s.less(is[i], install) })
    is = append(is, &Installation{})
    copy(is[i+1:], is[i:])
    is[i] = install
    return is
}

type installFilter struct {
    ProviderId   string
    Name         string
    IType        *v1.InstallationType `json:"type"`
}

type predicate func(i *Installation) bool

func parseFilter(filter installFilter) []predicate {
    var predicates []predicate
    // providerId
    if len(filter.ProviderId) > 0 {
        p := func(i *Installation) bool {
            if filter.ProviderId == i.ProviderId {
                return true
            }
            return false
        }
        predicates = append(predicates, p)
    }
    // name
    if len(filter.Name) > 0 {
        p := func(i *Installation) bool {
            if filter.Name == i.Name {
                return true
            }
            return false
        }
        predicates = append(predicates, p)
    }
    // IType
    if filter.IType != nil {
        p := func(i *Installation) bool {
            if *filter.IType == i.Type {
                return true
            }
            return false
        }
        predicates = append(predicates, p)
    }
    return predicates
}