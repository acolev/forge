package seeders

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBuildFixtureFromTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:fromtable?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	stmts := []string{
		`CREATE TABLE companies (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL,
			full_name TEXT,
			company_id INTEGER NOT NULL,
			is_active BOOLEAN,
			bio TEXT,
			created_at DATETIME,
			FOREIGN KEY (company_id) REFERENCES companies (id)
		)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	yaml, err := BuildFixtureFromTable(db, "users", 25)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	mustContain := []string{
		"table: users",
		"count: 25",
		`email: "fake:email"`,
		`full_name: "fake:full_name"`,
		`is_active: "fake:bool"`,
		`bio: "fake:sentence"`,
		`company_id: "ref:companies|id=1|id"`, // FK -> $ref
	}
	for _, want := range mustContain {
		if !strings.Contains(yaml, want) {
			t.Fatalf("generated YAML missing %q:\n%s", want, yaml)
		}
	}

	// Auto PK and timestamp columns must be omitted.
	for _, notWant := range []string{"  id:", "created_at:"} {
		if strings.Contains(yaml, notWant) {
			t.Fatalf("generated YAML should not contain %q:\n%s", notWant, yaml)
		}
	}
}

func TestBuildFixtureFromTableMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:fromtable_missing?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := BuildFixtureFromTable(db, "nope", 10); err == nil {
		t.Fatal("expected error for missing table")
	}
}
