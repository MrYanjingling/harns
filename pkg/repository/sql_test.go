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

	users := []Object{
		{
			"name":   "Zhang san",
			"age":    22,
			"labels": Object{"color": "red", "height": 180},
			"tags":   Array{"younger", "tall"},
		},
		{
			"name":   "Li si",
			"age":    25,
			"labels": Object{"color": "green", "weight": 120},
			"tags":   Array{"fashion"},
		},
		{
			"name":   "Wang wu",
			"age":    30,
			"labels": Object{"color": "blue", "height": 150},
			"tags":   Array{"older", "short"},
		},
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
		if foundUsers[0]["name"] != "Zhang san" {
			t.Errorf("Expected 'Zhang san' to be first, got %s", foundUsers[0]["name"])
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
		if foundUser["name"] != "Li si" {
			t.Errorf("Expected found 'Li si', got %s", foundUser["name"])
		}
	})

}

func createRepo(db *sql.DB) *repo {
	return &repo{
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
