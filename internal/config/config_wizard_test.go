package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAndParseDSN(t *testing.T) {
	dsn, err := BuildDSN(DSNParts{Driver: "postgres", Host: "db.local", Port: "6543", User: "bob", Password: "p@ss/word", DBName: "appdb"})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	p := ParseDSN(dsn)
	if p.Driver != "postgres" || p.Host != "db.local" || p.Port != "6543" || p.User != "bob" || p.DBName != "appdb" {
		t.Fatalf("round-trip mismatch: %+v (dsn=%s)", p, dsn)
	}
	if p.Password != "p@ss/word" {
		t.Fatalf("password not round-tripped: %q", p.Password)
	}

	sq, _ := BuildDSN(DSNParts{Driver: "sqlite", SQLitePath: "data/x.db"})
	if sq != "sqlite://data/x.db" {
		t.Fatalf("sqlite dsn = %q", sq)
	}
	if _, err := BuildDSN(DSNParts{Driver: "postgres", User: "u"}); err == nil {
		t.Fatal("expected error when dbname missing")
	}
}

func TestUpdateEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.forge")
	if err := os.WriteFile(path, []byte("# comment\nFORGE_DB_DSN=old\nAPP_SECRET=keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := UpdateEnvFile(path, map[string]string{
		"FORGE_DB_DSN":      "sqlite://x.db",
		"FORGE_PLUGINS_DIR": ".forge/plugins",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	out, _ := os.ReadFile(path)
	s := string(out)
	if !strings.Contains(s, "FORGE_DB_DSN=sqlite://x.db") {
		t.Fatalf("key not replaced:\n%s", s)
	}
	if !strings.Contains(s, "APP_SECRET=keep") {
		t.Fatalf("unrelated key dropped:\n%s", s)
	}
	if !strings.Contains(s, "# comment") {
		t.Fatalf("comment dropped:\n%s", s)
	}
	if !strings.Contains(s, "FORGE_PLUGINS_DIR=.forge/plugins") {
		t.Fatalf("new key not appended:\n%s", s)
	}
	if strings.Contains(s, "FORGE_DB_DSN=old") {
		t.Fatalf("old value not removed:\n%s", s)
	}
}

func TestRunWizardPiped(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	// driver, host, port, user, password, dbname, plugins, models dir, models pkg
	input := strings.Join([]string{
		"postgres", "", "", "bob", "secret", "appdb", "", "", "",
	}, "\n") + "\n"

	var out bytes.Buffer
	res, err := RunWizard(strings.NewReader(input), &out, nil)
	if err != nil {
		t.Fatalf("wizard: %v", err)
	}
	want := "postgres://bob:secret@localhost:5432/appdb"
	if res.DSN != want {
		t.Fatalf("DSN = %q, want %q", res.DSN, want)
	}

	saved := ReadEnvFileValue(".env.forge", ForgeDBDSNKey)
	if saved != want {
		t.Fatalf("saved DSN = %q, want %q", saved, want)
	}
	if ReadEnvFileValue(".env.forge", ForgeModelsPackageKey) != DefaultModelsPackage {
		t.Fatalf("models package default not written")
	}
}
