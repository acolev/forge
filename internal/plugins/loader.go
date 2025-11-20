package plugins

import (
	"encoding/json"
	"fmt"
	"forge/internal/hooks"
	"os"
	"os/exec"
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

		// На Linux используем пакет plugin (.so). На macOS и прочих — процессный режим.
		if runtime.GOOS == "linux" {
			loadPluginsFromRoot(rootCmd, root)
		} else if runtime.GOOS == "darwin" {
			loadPluginsFromRoot(rootCmd, root)
		} else {
			loadProcessPluginsFromRoot(rootCmd, root)
		}
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

func loadProcessPluginsFromRoot(rootCmd *cobra.Command, root string) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			fmt.Printf("error scanning plugin dir %s: %v\n", path, walkErr)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Ignore .so files (they're for native plugin.Open and not supported on macOS).
		if filepath.Ext(d.Name()) == ".so" {
			return nil
		}

		// Only consider executable files
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if fi.Mode()&0111 == 0 {
			return nil
		}

		if !isPluginForCurrentPlatform(d.Name()) {
			return nil
		}

		loadProcessPlugin(rootCmd, path)
		return nil
	})

	if err != nil {
		fmt.Printf("error walking plugin root %s: %v\n", root, err)
	}
}

func loadProcessPlugin(rootCmd *cobra.Command, path string) {
	cmd := exec.Command(path, "--describe")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("unable to describe plugin %s: %v\n", path, err)
		return
	}

	var desc struct {
		Name     string `json:"name"`
		Commands []struct {
			Use   string `json:"use"`
			Short string `json:"short"`
		} `json:"commands"`
	}

	if err := json.Unmarshal(out, &desc); err != nil {
		fmt.Printf("invalid plugin description from %s: %v\n", path, err)
		return
	}

	for _, c := range desc.Commands {
		sub := &cobra.Command{
			Use:   c.Use,
			Short: c.Short,
			RunE: func(p string) func(cmd *cobra.Command, args []string) error {
				return func(cmd *cobra.Command, args []string) error {
					// Запускаем плагин процессом, прокидываем аргументы
					execArgs := append([]string{p}, args...)
					execCmd := exec.Command(path, execArgs...)
					execCmd.Stdin = os.Stdin
					execCmd.Stdout = os.Stdout
					execCmd.Stderr = os.Stderr
					return execCmd.Run()
				}
			}(c.Use),
		}
		rootCmd.AddCommand(sub)
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
