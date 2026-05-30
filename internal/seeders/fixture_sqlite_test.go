package seeders

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestRunFixtureSQLite verifies that fixture seeding works on sqlite — i.e. it
// no longer crashes on the Postgres-only information_schema introspection, map
// values land in JSON columns, and on_conflict: update_all upserts correctly.
func TestRunFixtureSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT,
		name TEXT,
		meta TEXT
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := ensureTable(db); err != nil {
		t.Fatalf("ensure seeds table: %v", err)
	}

	// First insert — note the map value for the JSON-ish `meta` column, which
	// previously reached the driver as a raw map and failed to bind.
	seed := YAMLSeed{
		Name:        "users-fixture",
		Type:        "fixture",
		Table:       "users",
		OnConflict:  "update_all",
		ConflictKey: []string{"id"},
		Rows: []map[string]any{
			{"id": 1, "email": "a@example.com", "name": "Alice", "meta": map[string]any{"role": "admin"}},
			{"id": 2, "email": "b@example.com", "name": "Bob", "meta": map[string]any{"role": "user"}},
		},
	}
	if err := runFixture(db, seed, 1); err != nil {
		t.Fatalf("runFixture insert: %v", err)
	}

	var count int64
	if err := db.Table("users").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}

	// Second run with a different name but same ids — update_all must upsert,
	// not duplicate or error.
	seed2 := seed
	seed2.Name = "users-fixture-2"
	seed2.Rows = []map[string]any{
		{"id": 1, "email": "a@example.com", "name": "Alice Updated", "meta": map[string]any{"role": "owner"}},
	}
	if err := runFixture(db, seed2, 2); err != nil {
		t.Fatalf("runFixture upsert: %v", err)
	}

	if err := db.Table("users").Count(&count).Error; err != nil {
		t.Fatalf("count after upsert: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows after upsert, got %d", count)
	}

	var name string
	if err := db.Table("users").Select("name").Where("id = ?", 1).Scan(&name).Error; err != nil {
		t.Fatalf("scan name: %v", err)
	}
	if name != "Alice Updated" {
		t.Fatalf("expected upserted name 'Alice Updated', got %q", name)
	}
}
