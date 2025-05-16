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
	ctx := SqlContext{schema, "user", nil}
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

	t.Run("Insert some tests data", func(t *testing.T) {
		err := repo.Insert(ctx, users)
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

}

func createRepo(db *sql.DB) *SqlRepo {
	return &SqlRepo{
		sb: sqlBuilder{
			dialect: newDialectSqlite(),
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
    "$schema": "http://json-schema.org/draft-07/schema#",
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
