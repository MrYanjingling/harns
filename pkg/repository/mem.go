package repository

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type MemRepository struct {
	// resource name - id - object
	caches map[string]map[any]Object
}

type ResourceName string

func NewMemRepository() Repository[Object, ResourceName] {
	return &MemRepository{
		caches: make(map[string]map[any]Object),
	}
}

func (m *MemRepository) Find(ctx ResourceName, query *Query) ([]Object, error) {
	cache, ok := m.caches[string(ctx)]
	if !ok {
		return nil, nil
	}

	results := m.filterObjects(cache, query.Filters)
	results = m.applyProjection(results, query.Projection)
	results = m.applyDistinct(results, query.Projection, query.Distinct)
	results = m.applySorting(results, query.Sort)
	results = m.applyPagination(results, query.Skip, query.Limit)

	return results, nil
}

// filterObjects filters objects based on the provided filters
func (m *MemRepository) filterObjects(objects map[any]Object, filters Filters) []Object {
	var results []Object
	for _, item := range objects {
		if filters.MatchValue(item) {
			results = append(results, item)
		}
	}
	return results
}

// applyProjection applies the projection to the results
func (m *MemRepository) applyProjection(results []Object, projection map[string]bool) []Object {
	if projection == nil || len(projection) == 0 {
		return results
	}
	for i, item := range results {
		results[i] = m.projectObject(item, projection)
	}
	return results
}

// projectObject creates a new object with only the projected fields
func (m *MemRepository) projectObject(obj Object, projection map[string]bool) Object {
	newItem := Object{}
	for field := range projection {
		if val, ok := obj[field]; ok {
			newItem[field] = val
		}
	}
	return newItem
}

// applyDistinct applies the distinct operation to the results
func (m *MemRepository) applyDistinct(results []Object, projection map[string]bool, distinct bool) []Object {
	if !distinct {
		return results
	}
	seen := make(map[string]bool)
	var uniqueResults []Object
	for _, item := range results {
		key := buildDistinctKey(item, projection)
		if !seen[key] {
			seen[key] = true
			uniqueResults = append(uniqueResults, item)
		}
	}
	return uniqueResults
}

// applySorting sorts the results based on the provided sort criteria
func (m *MemRepository) applySorting(results []Object, sortCriteria map[string]SortOrder) []Object {
	if len(sortCriteria) == 0 {
		return results
	}
	sort.SliceStable(results, func(i, j int) bool {
		for field, order := range sortCriteria {
			valI, okI := results[i][field]
			valJ, okJ := results[j][field]
			if !okI && !okJ {
				continue
			}
			if !okI {
				return order == SortOrderAsc
			}
			if !okJ {
				return order == SortOrderDesc
			}
			if cmp := compareValues(valI, valJ); cmp != 0 {
				return order == SortOrderAsc && cmp < 0 || order == SortOrderDesc && cmp > 0
			}
		}
		return false
	})
	return results
}

// applyPagination applies skip and limit to the results
func (m *MemRepository) applyPagination(results []Object, skip, limit *int) []Object {
	s := 0
	if skip != nil {
		s = *skip
	}
	l := len(results)
	if limit != nil && *limit < l {
		l = *limit
	}
	if s > len(results) {
		return []Object{}
	}
	end := s + l
	if end > len(results) {
		end = len(results)
	}
	return results[s:end]
}

func (m *MemRepository) Insert(ctx ResourceName, items []Object) error {
	if _, ok := m.caches[string(ctx)]; !ok {
		m.caches[string(ctx)] = make(map[any]Object)
	}

	for _, obj := range items {
		id, ok := obj["id"]
		if !ok {
			return errors.New("missing id field")
		}
		m.caches[string(ctx)][id] = obj
	}
	return nil
}

func (m *MemRepository) Update(ctx ResourceName, fn UpdateFn[Object], filters *Filters) ([]Object, error) {
	cache, ok := m.caches[string(ctx)]
	if !ok {
		return nil, nil
	}

	var updatedItems []Object
	for id, obj := range cache {
		if filters.MatchValue(obj) {
			newItem := fn(&obj)
			cache[id] = newItem
			updatedItems = append(updatedItems, newItem)
		}
	}
	return updatedItems, nil
}

func (m *MemRepository) Clear(ctx ResourceName, filters *Filters) error {
	cache, ok := m.caches[string(ctx)]
	if !ok {
		return nil
	}

	for id, obj := range cache {
		if filters.MatchValue(obj) {
			delete(cache, id)
		}
	}
	return nil
}

func (m *MemRepository) Count(ctx ResourceName, query *Query) (uint64, error) {
	cache, ok := m.caches[string(ctx)]
	if !ok {
		return 0, nil
	}

	var count uint64
	for _, obj := range cache {
		if query.Filters.MatchValue(obj) {
			count++
		}
	}
	return count, nil
}

func buildDistinctKey(item Object, projection map[string]bool) string {
	var keys []string
	for k := range item {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k + ":" + fmt.Sprintf("%v|", item[k]))
	}
	return sb.String()
}

func compareValues(a, b any) int {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a - b
		}
	case float64:
		switch b := b.(type) {
		case float64:
			switch {
			case a < b:
				return -1
			case a > b:
				return 1
			default:
				return 0
			}
		}
	case string:
		switch b := b.(type) {
		case string:
			return strings.Compare(a, b)
		}
	}
	return 0
}
