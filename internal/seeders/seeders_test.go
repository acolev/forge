package seeders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandFixtureRowsFromTemplate(t *testing.T) {
	t.Parallel()

	rows, err := expandFixtureRows(YAMLSeed{
		Table: "users",
		Count: 3,
		Template: map[string]any{
			"name":      "fake:full_name",
			"email":     "fake:email",
			"is_active": "fake:bool",
			"age":       "fake:int:18:65",
		},
	})
	if err != nil {
		t.Fatalf("expandFixtureRows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows len = %d, want 3", len(rows))
	}
	for _, row := range rows {
		if _, ok := row["name"].(string); !ok {
			t.Fatalf("name should be string: %#v", row["name"])
		}
		if _, ok := row["email"].(string); !ok {
			t.Fatalf("email should be string: %#v", row["email"])
		}
		if _, ok := row["is_active"].(bool); !ok {
			t.Fatalf("is_active should be bool: %#v", row["is_active"])
		}
		if age, ok := row["age"].(int); !ok || age < 18 || age > 65 {
			t.Fatalf("age should be int in range: %#v", row["age"])
		}
	}
}

func TestCreateSeedFixtureTemplate(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	path, err := CreateSeed("users", "fixture")
	if err != nil {
		t.Fatalf("CreateSeed: %v", err)
	}

	if filepath.Dir(path) != filepath.Join(".", "database", "seeds") && !strings.Contains(path, "database/seeds") {
		t.Fatalf("unexpected path: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read seed file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "type: fixture") {
		t.Fatalf("missing fixture type in seed file: %s", text)
	}
	if !strings.Contains(text, "fake:email") {
		t.Fatalf("missing fake template example in seed file: %s", text)
	}
}
