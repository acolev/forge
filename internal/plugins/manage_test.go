package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePluginSlug(t *testing.T) {
	t.Parallel()

	vendor, name, err := parsePluginSlug("bookly/migrate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vendor != "bookly" || name != "migrate" {
		t.Fatalf("got %s/%s", vendor, name)
	}
}

func TestCreatePluginScaffold(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginDir, err := CreatePluginScaffold(root, "bookly", "migrate", "")
	if err != nil {
		t.Fatalf("CreatePluginScaffold: %v", err)
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "plugin.json")); err != nil {
		t.Fatalf("missing plugin.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "src", "main.go")); err != nil {
		t.Fatalf("missing src/main.go: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "src", "go.mod")); err != nil {
		t.Fatalf("missing src/go.mod: %v", err)
	}
}

func TestCreatePluginScaffoldFailsWhenPluginExists(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := CreatePluginScaffold(root, "bookly", "migrate", ""); err != nil {
		t.Fatalf("first CreatePluginScaffold: %v", err)
	}

	if _, err := CreatePluginScaffold(root, "bookly", "migrate", ""); err == nil {
		t.Fatal("expected error for existing plugin, got nil")
	}
}

func TestCreatePluginScaffoldWithHook(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginDir, err := CreatePluginScaffold(root, "bookly", "migrate", "db.migrate.before")
	if err != nil {
		t.Fatalf("CreatePluginScaffold: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(pluginDir, "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	if !strings.Contains(string(content), `"db.migrate.before"`) {
		t.Fatalf("plugin.json does not contain hook: %s", string(content))
	}
	if strings.Contains(string(content), `"ping"`) {
		t.Fatalf("hook scaffold should not include default ping command: %s", string(content))
	}

	mainContent, err := os.ReadFile(filepath.Join(pluginDir, "src", "main.go"))
	if err != nil {
		t.Fatalf("read src/main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), "AutoMigrate") {
		t.Fatalf("hook scaffold should include AutoMigrate placeholder: %s", string(mainContent))
	}
}
