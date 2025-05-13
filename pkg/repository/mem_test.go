package repository

import (
	"testing"
)

func TestMemRepository(t *testing.T) {
	repo := NewMemRepository()
	ctx := ResourceName("test")

	// 测试 Insert
	items := []Object{
		{"_id": 1, "name": "Alice", "age": 25, "tags": []string{"a", "b"}},
		{"_id": 2, "name": "Bob", "age": 30, "tags": []string{"b", "c"}},
		{"_id": 3, "name": "Charlie", "age": 35, "tags": []string{"c", "d"}},
	}
	if err := repo.Insert(ctx, items); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// 测试 Update
	t.Run("Update", func(t *testing.T) {
		updateFn := func(item *Object) Object {
			(*item)["age"] = 26
			return *item
		}
		filters := Filters{FieldFilter{Key: "name", Op: EQ{}, Value: "Alice"}}
		updatedItems, err := repo.Update(ctx, updateFn, &filters)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if len(updatedItems) != 1 {
			t.Errorf("Expected 1 updated item, got %d", len(updatedItems))
		}
	})

	// 测试 Clear
	t.Run("Clear", func(t *testing.T) {
		filters := Filters{FieldFilter{Key: "name", Op: EQ{}, Value: "Bob"}}
		if err := repo.Clear(ctx, &filters); err != nil {
			t.Fatalf("Clear failed: %v", err)
		}
		count, _ := repo.Count(ctx, NewQuery())
		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}
	})

	// 测试 Find
	t.Run("Find", func(t *testing.T) {
		query := NewQuery()
		query.Filters = Filters{FieldFilter{Key: "age", Op: GTE{}, Value: 25}}
		results, err := repo.Find(ctx, query)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})
}

// 拆分后的独立测试用例
func TestFind_Filters(t *testing.T) {
	repo := NewMemRepository()
	ctx := ResourceName("test")

	// 初始化数据
	items := []Object{
		{"_id": 1, "name": "Alice", "age": 25, "tags": []string{"a", "b"}},
		{"_id": 2, "name": "Bob", "age": 30, "tags": []string{"b", "c"}},
		{"_id": 3, "name": "Charlie", "age": 35, "tags": []string{"c", "d"}},
	}
	if err := repo.Insert(ctx, items); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	t.Run("NestedFilters", func(t *testing.T) {
		query := NewQuery()
		query.Filters = Filters{
			FieldFilter{Key: "age", Op: GTE{}, Value: 30},
			AndFilter{
				Filters: Filters{
					FieldFilter{Key: "name", Op: EQ{}, Value: "Charlie"},
					FieldFilter{Key: "tags", Op: CONTAINS{}, Value: "d"},
				},
			},
		}
		results, err := repo.Find(ctx, query)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})
}

func TestFind_Projection(t *testing.T) {
	repo := NewMemRepository()
	ctx := ResourceName("test")

	// 初始化相同数据...

	t.Run("FieldProjection", func(t *testing.T) {
		query := NewQuery()
		query.Projection = map[string]bool{"name": true, "age": true}
		results, _ := repo.Find(ctx, query)
		for _, item := range results {
			if len(item) != 2 {
				t.Errorf("Projection failed: expected 2 fields, got %d", len(item))
			}
		}
	})
}

// 新增的测试用例
func TestFind_Operators(t *testing.T) {
	repo := NewMemRepository()
	ctx := ResourceName("test")

	// 测试数据准备
	items := []Object{
		{"_id": 1, "name": "TestA", "score": 85, "tags": []string{"x", "y"}},
		{"_id": 2, "name": "TestB", "score": 92, "tags": []string{"y", "z"}},
	}
	repo.Insert(ctx, items)

	t.Run("LT_Operator", func(t *testing.T) {
		query := NewQuery()
		query.Filters = Filters{FieldFilter{Key: "score", Op: LT{}, Value: 90}}
		results, _ := repo.Find(ctx, query)
		if len(results) != 1 {
			t.Errorf("LT operator test failed")
		}
	})

	t.Run("REGEXP_Operator", func(t *testing.T) {
		query := NewQuery()
		query.Filters = Filters{FieldFilter{Key: "name", Op: REGEXP{}, Value: "^Test"}}
		results, _ := repo.Find(ctx, query)
		if len(results) != 2 {
			t.Errorf("REGEXP operator test failed")
		}
	})
}

func TestFind_Distinct(t *testing.T) {
	repo := NewMemRepository()
	ctx := ResourceName("test")

	// 测试数据准备
	items := []Object{
		{"_id": 1, "name": "Alice", "city": "Beijing"},
		{"_id": 2, "name": "Bob", "city": "Shanghai"},
		{"_id": 3, "name": "Alice", "city": "Beijing"},
	}
	repo.Insert(ctx, items)

	t.Run("DistinctByName", func(t *testing.T) {
		query := NewQuery()
		query.Distinct = true
		query.Projection = map[string]bool{"name": true}
		results, _ := repo.Find(ctx, query)
		if len(results) != 2 {
			t.Errorf("Distinct test failed, expected 2 unique names")
		}
	})
}

func TestFind_SortPagination(t *testing.T) {
	repo := NewMemRepository()
	ctx := ResourceName("test")

	// 测试数据准备
	items := []Object{
		{"_id": 1, "name": "A", "score": 80},
		{"_id": 2, "name": "B", "score": 90},
		{"_id": 3, "name": "C", "score": 70},
	}
	repo.Insert(ctx, items)

	t.Run("MultiFieldSort", func(t *testing.T) {
		query := NewQuery()
		query.Sort = map[string]SortOrder{
			"score": SortOrderDesc,
			"name":  SortOrderAsc,
		}
		results, _ := repo.Find(ctx, query)
		if results[0]["name"] != "B" || results[1]["name"] != "A" || results[2]["name"] != "C" {
			t.Errorf("Multi-field sort failed")
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		query := NewQuery()
		query.Sort = map[string]SortOrder{"score": SortOrderDesc}
		query.Skip = intPtr(1)
		query.Limit = intPtr(1)
		results, _ := repo.Find(ctx, query)
		if results[0]["name"] != "A" {
			t.Errorf("Pagination failed")
		}
	})
}

// Helper function to create pointer to int
func intPtr(i int) *int {
	return &i
}
