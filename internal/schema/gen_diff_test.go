package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGoModels(t *testing.T) {
	db := openTestDB(t)
	m, err := Introspect(db)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	code := RenderGoModels(m, "models", []string{"users"})
	for _, want := range []string{
		"package models",
		"type User struct {",
		"ID",
		"uint",
		"primaryKey",
		`func (User) TableName() string { return "users" }`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
	// posts was not requested — must be absent.
	if strings.Contains(code, "type Post struct") {
		t.Fatalf("did not expect Post struct:\n%s", code)
	}
}

func TestSnapshotRoundTripAndDiff(t *testing.T) {
	old := &Model{Driver: "sqlite", Tables: []Table{
		{
			Name:       "users",
			Columns:    []Column{{Name: "id", Type: "integer"}, {Name: "email", Type: "text", Nullable: false}},
			PrimaryKey: []string{"id"},
		},
		{Name: "legacy", Columns: []Column{{Name: "id", Type: "integer"}}},
	}}

	// Round-trip through JSON.
	data, err := SnapshotJSON(old)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	path := filepath.Join(t.TempDir(), "snap.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Tables) != 2 {
		t.Fatalf("round-trip lost tables: %+v", loaded.Tables)
	}

	// New model: drop "legacy", add "posts", change users.email nullability, add a column.
	newM := &Model{Driver: "sqlite", Tables: []Table{
		{
			Name: "users",
			Columns: []Column{
				{Name: "id", Type: "integer"},
				{Name: "email", Type: "text", Nullable: true},
				{Name: "phone", Type: "text"},
			},
			PrimaryKey: []string{"id"},
		},
		{Name: "posts", Columns: []Column{{Name: "id", Type: "integer"}}},
	}}

	d := DiffModels(loaded, newM)
	if d.Empty() {
		t.Fatal("expected differences")
	}
	out := RenderDiff(d)
	for _, want := range []string{
		"+ table posts",
		"- table legacy",
		"~ table users",
		"+ column phone",
		"~ column email: nullable",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("diff output missing %q:\n%s", want, out)
		}
	}
}

func TestDiffEmptyWhenEqual(t *testing.T) {
	m := &Model{Driver: "sqlite", Tables: []Table{
		{Name: "users", Columns: []Column{{Name: "id", Type: "integer"}}, PrimaryKey: []string{"id"}},
	}}
	if !DiffModels(m, m).Empty() {
		t.Fatal("identical models should diff empty")
	}
}
