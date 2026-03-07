package migrations

import (
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunMigrationsRollsBackBatchOnFailure(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	migrationsDir := filepath.Join(tempDir, "database", "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	firstMigration := "-- UP\nCREATE TABLE users (id INTEGER PRIMARY KEY);\n-- DOWN\nDROP TABLE users;\n"
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_create_users.sql"), []byte(firstMigration), 0o644); err != nil {
		t.Fatalf("write first migration: %v", err)
	}

	secondMigration := "-- UP\nCREATE TABLE users (id INTEGER PRIMARY KEY);\n-- DOWN\nDROP TABLE users;\n"
	if err := os.WriteFile(filepath.Join(migrationsDir, "002_create_users_again.sql"), []byte(secondMigration), 0o644); err != nil {
		t.Fatalf("write second migration: %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&Migration{}); err != nil {
		t.Fatalf("auto migrate metadata: %v", err)
	}

	err = RunMigrations(db)
	if err == nil {
		t.Fatal("expected migration error, got nil")
	}

	if db.Migrator().HasTable("users") {
		t.Fatal("expected users table to be rolled back")
	}

	var appliedCount int64
	if err := db.Model(&Migration{}).Count(&appliedCount).Error; err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if appliedCount != 0 {
		t.Fatalf("expected no recorded migrations after rollback, got %d", appliedCount)
	}
}

func TestResolveStubAndPlaceholder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		wantStub        string
		wantPlaceholder string
	}{
		{
			name:            "explicit create table prefix",
			input:           "create_table_users",
			wantStub:        "create_table",
			wantPlaceholder: "users",
		},
		{
			name:            "implicit create fallback",
			input:           "create_user_data",
			wantStub:        "create_table",
			wantPlaceholder: "user_data",
		},
		{
			name:            "implicit update fallback",
			input:           "update_user_data",
			wantStub:        "update_table",
			wantPlaceholder: "user_data",
		},
		{
			name:            "unknown prefix",
			input:           "custom_report_refresh",
			wantStub:        "",
			wantPlaceholder: "custom_report_refresh",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotStub, gotPlaceholder := resolveStubAndPlaceholder(tt.input)
			if gotStub != tt.wantStub {
				t.Fatalf("stub = %q, want %q", gotStub, tt.wantStub)
			}
			if gotPlaceholder != tt.wantPlaceholder {
				t.Fatalf("placeholder = %q, want %q", gotPlaceholder, tt.wantPlaceholder)
			}
		})
	}
}
