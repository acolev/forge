package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentSettingsPrefersDotEnvForge(t *testing.T) {
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

	t.Setenv(ForgeDBDSNKey, "")
	t.Setenv(ForgePluginsDirKey, "")

	if err := os.WriteFile(FallbackEnvFile, []byte(ForgeDBDSNKey+"=sqlite://fallback.db\n"), 0o644); err != nil {
		t.Fatalf("write fallback env: %v", err)
	}
	if err := os.WriteFile(DefaultEnvFile, []byte(ForgeDBDSNKey+"=sqlite://primary.db\n"+ForgePluginsDirKey+"=.forge/custom\n"), 0o644); err != nil {
		t.Fatalf("write primary env: %v", err)
	}

	settings, err := CurrentSettings()
	if err != nil {
		t.Fatalf("CurrentSettings: %v", err)
	}

	if settings.EnvFile != DefaultEnvFile {
		t.Fatalf("EnvFile = %q, want %q", settings.EnvFile, DefaultEnvFile)
	}
	if settings.DBDSN != "sqlite://primary.db" {
		t.Fatalf("DBDSN = %q, want %q", settings.DBDSN, "sqlite://primary.db")
	}
	if settings.PluginsDir != ".forge/custom" {
		t.Fatalf("PluginsDir = %q, want %q", settings.PluginsDir, ".forge/custom")
	}
}

func TestResolvePluginsDir(t *testing.T) {
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

	t.Setenv(ForgeDBDSNKey, "")
	t.Setenv(ForgePluginsDirKey, "")

	if err := os.WriteFile(DefaultEnvFile, []byte(ForgePluginsDirKey+"=.forge/plugins\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	got, err := ResolvePluginsDir("/workspace/project")
	if err != nil {
		t.Fatalf("ResolvePluginsDir: %v", err)
	}

	want := filepath.Join("/workspace/project", ".forge/plugins")
	if got != want {
		t.Fatalf("plugins dir = %q, want %q", got, want)
	}
}
