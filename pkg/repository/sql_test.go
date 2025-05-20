package repository

import (
	"database/sql"
	"github.com/bytedance/sonic"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"testing"
)

func TestFind(t *testing.T) {
	db := createDb()
	defer db.Close()

	repo := createRepo(db)
	schema := userSchema()
	ctx := Context{schema, "user"}
	mustNil(repo.Init(ctx))

	users := []Record{
		Record(Object{
			"name":   "Zhang san",
			"age":    22,
			"labels": Object{"color": "red", "height": 180},
			"tags":   Array{"younger", "tall"},
		}),
		Record(Object{
			"name":   "Li si",
			"age":    25,
			"labels": Object{"color": "green", "weight": 120},
			"tags":   Array{"fashion"},
		}),
		Record(Object{
			"name":   "Wang wu",
			"age":    30,
			"labels": Object{"color": "blue", "height": 150},
			"tags":   Array{"older", "short"},
		}),
	}

	t.Run("Create some tests data", func(t *testing.T) {
		err := repo.Create(ctx, users)
		mustNil(err)
	})

	t.Run("Find", func(t *testing.T) {
		foundUsers, err := repo.Find(ctx, &Query{})
		mustNil(err)
		if len(foundUsers) != len(users) {
			t.Errorf("Expected %d results, got %d", len(users), len(foundUsers))
		}
	})

	t.Run("Find by age greater than 25", func(t *testing.T) {
		query := &Query{
			Filter: &Binary{"age", &Gt{Value: 25}},
		}
		foundUsers, err := repo.Find(ctx, query)
		mustNil(err)
		if len(foundUsers) != 1 {
			t.Errorf("Expected 1 result, got %d", len(foundUsers))
		}
	})

	t.Run("Find and sort by age ascending", func(t *testing.T) {
		query := &Query{
			Sort: map[string]SortOrder{
				"age": SortOrderAsc,
			},
		}
		foundUsers, err := repo.Find(ctx, query)
		mustNil(err)
		if len(foundUsers) != len(users) {
			t.Errorf("Expected %d results, got %d", len(users), len(foundUsers))
		}
		if val, ok := foundUsers[0].Get("name"); !ok || val != "Zhang san" {
			t.Errorf("Expected 'Zhang san' to be first, got %s", val)
		}
	})

	t.Run("Find with pagination (limit 2)", func(t *testing.T) {
		query := &Query{
			Limit: 2,
		}
		foundUsers, err := repo.Find(ctx, query)
		mustNil(err)
		if len(foundUsers) != 2 {
			t.Errorf("Expected 2 results, got %d", len(foundUsers))
		}
	})

	t.Run("Find by json key", func(t *testing.T) {
		query := &Query{
			Filter: &Binary{Key: "labels.color", Op: &Eq{Value: "green"}},
		}
		foundUsers, err := repo.Find(ctx, query)
		mustNil(err)
		if len(foundUsers) != 1 {
			t.Errorf("Expected 1 results, got %d", len(foundUsers))
		}
		foundUser := foundUsers[0]
		if val, ok := foundUser.Get("name"); !ok || val != "Li si" {
			t.Errorf("Expected found 'Li si', got %s", val)
		}
	})

}

func TestUpdate(t *testing.T) {
	db := createDb()
	defer db.Close()

	repo := createRepo(db)
	schema := userSchema()
	ctx := Context{schema, "user"}
	mustNil(repo.Init(ctx))

	users := []Record{
		Record(Object{
			"name":   "Zhang san",
			"age":    22,
			"labels": Object{"color": "red", "height": 180},
			"tags":   Array{"younger", "tall"},
		}),
		Record(Object{
			"name":   "Li si",
			"age":    25,
			"labels": Object{"color": "green", "weight": 120},
			"tags":   Array{"fashion"},
		}),
		Record(Object{
			"name":   "Wang wu",
			"age":    30,
			"labels": Object{"color": "blue", "height": 150},
			"tags":   Array{"older", "short"},
		}),
	}

	t.Run("Create some tests data", func(t *testing.T) {
		err := repo.Create(ctx, users)
		mustNil(err)
	})

	t.Run("Update age of Zhang san to 23", func(t *testing.T) {
		query := &Query{
			Filter: &Binary{"name", &Eq{Value: "Zhang san"}},
		}
		updateFn := func(item Record) (Record, error) {
			item.Set("age", 23)
			return item, nil
		}
		updated, err := repo.Update(ctx, updateFn, query.Filter)
		mustNil(err)
		if val, ok := updated.Get("age"); !ok || val.(int) != 23 {
			t.Errorf("Expected age 23, got %v", val)
		}
	})

	t.Run("Update labels of Li si", func(t *testing.T) {
		query := &Query{
			Filter: &Binary{"name", &Eq{Value: "Li si"}},
		}
		updateFn := func(item Record) (Record, error) {
			item.Set("labels", Object{"color": "yellow", "weight": 125})
			return item, nil
		}
		_, err := repo.Update(ctx, updateFn, query.Filter)
		mustNil(err)
	})

	t.Run("Update tags of Wang wu", func(t *testing.T) {
		query := &Query{
			Filter: &Binary{"name", &Eq{Value: "Wang wu"}},
		}
		updateFn := func(item Record) (Record, error) {
			item.Set("tags", Array{"middle", "short"})
			return item, nil
		}
		_, err := repo.Update(ctx, updateFn, query.Filter)
		mustNil(err)
	})

	t.Run("Verify updates", func(t *testing.T) {
		foundUsers, err := repo.Find(ctx, &Query{})
		mustNil(err)
		if len(foundUsers) != len(users) {
			t.Errorf("Expected %d results, got %d", len(users), len(foundUsers))
		}

		for _, user := range foundUsers {
			name, _ := user.Get("name")
			switch name {
			case "Zhang san":
				if age, ok := user.Get("age"); !ok || age != 23 {
					t.Errorf("Expected age 23 for Zhang san, got %v", age)
				}
			case "Li si":
				labels, _ := user.Get("labels")
				if labels.(Object)["color"] != "yellow" {
					t.Errorf("Expected color yellow for Li si, got %v", labels.(Object)["color"])
				}
			case "Wang wu":
				tags, _ := user.Get("tags")
				if tags.(Array)[0] != "middle" {
					t.Errorf("Expected tag middle for Wang wu, got %v", tags.(Array)[0])
				}
			}
		}
	})
}

func createRepo(db *sql.DB) *sqlRepo {
	return &sqlRepo{
		sb: sqlBuilder{
			dialect: dialectSqlite{},
		},
		db: db,
	}
}

func createDb() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	mustNil(err)

	err = db.Ping()
	mustNil(err)
	return db
}

func mustNil(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func userSchema() Schema {
	node, err := sonic.Get([]byte(`
{
    "$Schema": "http://json-Schema.org/draft-07/Schema#",
    "title": "user",
    "type": "object",
    "properties": {
        "id": {
            "type": "integer",
            "readOnly": true
        },
		"age": {
            "type": "integer"
        },
        "name": {
            "type": "string"
        },
        "labels": {
            "type": "object"
        },
        "tags": {
            "type": "array"
        }
    }
}
`))

	mustNil(err)

	return JsonSchema{node}
}
