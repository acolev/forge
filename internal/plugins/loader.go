package plugins

import (
	"fmt"
	"forge/internal/migrations"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func LoadPlugins(rootCmd *cobra.Command) {
	roots := discoverPluginRoots()

	seenRoots := make(map[string]struct{})
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if _, ok := seenRoots[root]; ok {
			continue
		}
		seenRoots[root] = struct{}{}

		loadPluginsFromRoot(rootCmd, root)
	}
}

func discoverPluginRoots() []string {
	var dirs []string

	// 1) Explicit override via environment variable.
	if envDir := os.Getenv("FORGE_PLUGINS_DIR"); envDir != "" {
		dirs = append(dirs, envDir)
	}

	// 2) Project-local directory: .forge/plugins
	dirs = append(dirs, filepath.Join(".forge", "plugins"))

	// 3) Legacy directory: database/plugins
	dirs = append(dirs, filepath.Join("database", "plugins"))

	// 4) Global directory: $HOME/.forge/plugins
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".forge", "plugins"))
	}

	return dirs
}

func loadPluginsFromRoot(rootCmd *cobra.Command, root string) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		// Not a directory or does not exist — skip silently.
		return
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Do not stop scanning because of one error.
			fmt.Printf("error scanning plugin dir %s: %v\n", path, walkErr)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(d.Name()) != ".so" {
			return nil
		}

		if !isPluginForCurrentPlatform(d.Name()) {
			return nil
		}

		loadSinglePlugin(rootCmd, path)
		return nil
	})

	if err != nil {
		fmt.Printf("error walking plugin root %s: %v\n", root, err)
	}
}

func isPluginForCurrentPlatform(filename string) bool {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(name, "_")

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch len(parts) {
	case 1:
		// "example" -> matches all
		return true
	case 2:
		// "example_linux" -> OS only
		return parts[1] == goos
	case 3:
		// "example_darwin_arm64" -> OS + ARCH
		return parts[1] == goos && parts[2] == goarch
	default:
		// More segments than we expect -> ignore
		return false
	}
}

func loadSinglePlugin(rootCmd *cobra.Command, path string) {
	p, err := plugin.Open(path)
	if err != nil {
		fmt.Printf("unable to open plugin %s: %v\n", path, err)
		return
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		fmt.Printf("unable to find Plugin symbol in %s: %v\n", path, err)
		return
	}

	pluginInstance, ok := sym.(Plugin)
	if !ok {
		fmt.Printf("invalid Plugin type in %s\n", path)
		return
	}

	if mh, ok := pluginInstance.(migrations.MigrationHook); ok {
		migrations.RegisterHook(mh)
	}

	pluginInstance.RegisterCommands(rootCmd)
	// fmt.Printf("Loaded plugin: %s (%s)\n", pluginInstance.Name(), path)
}
