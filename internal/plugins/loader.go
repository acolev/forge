package plugins

import (
	"fmt"
	"forge/internal/hooks"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
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

	if envDir := os.Getenv("FORGE_PLUGINS_DIR"); envDir != "" {
		dirs = append(dirs, envDir)
	}

	dirs = append(dirs, filepath.Join(".forge", "plugins"))

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

	var pluginInstance Plugin

	// Try several possible shapes of the exported symbol:
	// 1) the symbol itself implements the Plugin interface
	// 2) the symbol is a pointer to an interface variable (*Plugin)
	// 3) the symbol is a pointer to the variable holding the concrete plugin
	if pi, ok := sym.(Plugin); ok {
		pluginInstance = pi
	} else if pptr, ok := sym.(*Plugin); ok && pptr != nil {
		// symbol is pointer to an interface variable
		pluginInstance = *pptr
	} else {
		// Use reflection to handle cases like: var Plugin = &ExamplePlugin{}
		v := reflect.ValueOf(sym)
		if v.Kind() == reflect.Ptr && v.Elem().IsValid() {
			val := v.Elem()
			// If the element implements Plugin, use it
			pluginIFace := reflect.TypeOf((*Plugin)(nil)).Elem()
			if val.Type().Implements(pluginIFace) {
				pluginInstance = val.Interface().(Plugin)
			} else if val.Kind() == reflect.Ptr && val.Elem().Type().Implements(pluginIFace) {
				// double pointer cases
				pluginInstance = val.Elem().Interface().(Plugin)
			}
		}
	}

	if pluginInstance == nil {
		fmt.Printf("invalid Plugin type in %s\n", path)
		return
	}

	// If plugin implements hooks.Handler, register it to receive events.
	if hh, ok := pluginInstance.(hooks.Handler); ok {
		hooks.Register(hh)
	}

	pluginInstance.RegisterCommands(rootCmd)
	// fmt.Printf("Loaded plugin: %s (%s)\n", pluginInstance.Name(), path)
}
