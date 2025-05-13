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

func NewMemRepository() *MemRepository {
	return &MemRepository{
		caches: make(map[string]map[any]Object),
	}
}

func (m MemRepository) Find(ctx ResourceName, query *Query) ([]Object, error) {
	cache, ok := m.caches[string(ctx)]
	if !ok {
		return nil, nil
	}

	var results []Object
	for _, item := range cache {
		if query.Filters.MatchValue(item) {
			results = append(results, item)
		}
	}

	// Apply Projection
	if query.Projection != nil && len(query.Projection) > 0 {
		for i := range results {
			newItem := Object{}
			for field := range query.Projection {
				if val, ok := results[i][field]; ok {
					newItem[field] = val
				}
			}
			results[i] = newItem
		}
	}

	// Apply Distinct
	if query.Distinct {
		seen := make(map[string]bool)
		var uniqueResults []Object
		for _, item := range results {
			key := buildDistinctKey(item, query.Projection)
			if !seen[key] {
				seen[key] = true
				uniqueResults = append(uniqueResults, item)
			}
		}
		results = uniqueResults
	}

	// Apply Sort
	if len(query.Sort) > 0 {
		sort.SliceStable(results, func(i, j int) bool {
			for field, order := range query.Sort {
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
	}

	// Apply Skip and Limit
	skip := 0
	if query.Skip != nil {
		skip = *query.Skip
	}
	limit := len(results)
	if query.Limit != nil {
		limit = *query.Limit
	}
	end := skip + limit
	if end > len(results) {
		end = len(results)
	}
	if skip > len(results) {
		results = []Object{}
	} else {
		results = results[skip:end]
	}

	return results, nil
}

func (m MemRepository) Insert(ctx ResourceName, items []Object) error {
	if _, ok := m.caches[string(ctx)]; !ok {
		m.caches[string(ctx)] = make(map[any]Object)
	}

	for _, obj := range items {
		id, ok := obj["_id"]
		if !ok {
			return errors.New("missing _id field")
		}
		m.caches[string(ctx)][id] = obj
	}
	return nil
}

func (m MemRepository) Update(ctx ResourceName, fn UpdateFn[Object], filters *Filters) ([]Object, error) {
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

func (m MemRepository) Clear(ctx ResourceName, filters *Filters) error {
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

func (m MemRepository) Count(ctx ResourceName, query *Query) (int32, error) {
	cache, ok := m.caches[string(ctx)]
	if !ok {
		return 0, nil
	}

	var count int32
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
