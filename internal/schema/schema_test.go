package schema

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Unique in-memory DB per test so shared-cache state does not leak between tests.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	stmts := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL,
			name TEXT
		)`,
		`CREATE UNIQUE INDEX ux_users_email ON users (email)`,
		`CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	return db
}

func TestIntrospectSQLite(t *testing.T) {
	db := openTestDB(t)

	m, err := Introspect(db)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}
	if m.Driver != "sqlite" {
		t.Fatalf("driver = %q", m.Driver)
	}
	if len(m.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d: %+v", len(m.Tables), m.Tables)
	}

	users := m.Table("users")
	if users == nil {
		t.Fatal("users table missing")
	}
	if len(users.PrimaryKey) != 1 || users.PrimaryKey[0] != "id" {
		t.Fatalf("users PK = %v", users.PrimaryKey)
	}
	var emailUnique bool
	for _, ix := range users.Indexes {
		if ix.Name == "ux_users_email" && ix.Unique && len(ix.Columns) == 1 && ix.Columns[0] == "email" {
			emailUnique = true
		}
	}
	if !emailUnique {
		t.Fatalf("expected unique index on users.email, indexes = %+v", users.Indexes)
	}

	posts := m.Table("posts")
	if posts == nil {
		t.Fatal("posts table missing")
	}
	if len(posts.ForeignKeys) != 1 {
		t.Fatalf("expected 1 FK on posts, got %+v", posts.ForeignKeys)
	}
	fk := posts.ForeignKeys[0]
	if fk.RefTable != "users" || fk.Columns[0] != "user_id" || fk.RefColumns[0] != "id" {
		t.Fatalf("unexpected FK: %+v", fk)
	}
}

func TestRenderMermaid(t *testing.T) {
	db := openTestDB(t)
	m, err := Introspect(db)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	out := RenderMermaid(m)
	if !strings.HasPrefix(out, "erDiagram") {
		t.Fatalf("missing erDiagram header:\n%s", out)
	}
	// PK/FK markers and the relationship line.
	for _, want := range []string{"users {", "posts {", "PK", "FK", "posts }o--|| users"} {
		if !strings.Contains(out, want) {
			t.Fatalf("mermaid output missing %q:\n%s", want, out)
		}
	}
}

func TestDumpSQLSQLite(t *testing.T) {
	db := openTestDB(t)
	m, err := Introspect(db)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}
	ddl, err := DumpSQL(db, m)
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	for _, want := range []string{"CREATE TABLE users", "CREATE TABLE posts", "ux_users_email"} {
		if !strings.Contains(ddl, want) {
			t.Fatalf("dump missing %q:\n%s", want, ddl)
		}
	}
}

func TestDropAllTables(t *testing.T) {
	db := openTestDB(t)
	if err := DropAllTables(db); err != nil {
		t.Fatalf("drop all: %v", err)
	}
	m, err := Introspect(db)
	if err != nil {
		t.Fatalf("introspect after drop: %v", err)
	}
	if len(m.Tables) != 0 {
		t.Fatalf("expected 0 tables after drop, got %d", len(m.Tables))
	}
}
