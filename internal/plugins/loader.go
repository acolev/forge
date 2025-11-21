package plugins

import (
	"encoding/json"
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
		// not fatal yet — попробуем альтернативные экспортированные символы ниже
	} else {
		// existing Plugin symbol handling
		var pluginInstance Plugin
		if pi, ok := sym.(Plugin); ok {
			pluginInstance = pi
		} else if pptr, ok := sym.(*Plugin); ok && pptr != nil {
			pluginInstance = *pptr
		} else {
			v := reflect.ValueOf(sym)
			if v.Kind() == reflect.Ptr && v.Elem().IsValid() {
				val := v.Elem()
				pluginIFace := reflect.TypeOf((*Plugin)(nil)).Elem()
				if val.Type().Implements(pluginIFace) {
					pluginInstance = val.Interface().(Plugin)
				} else if val.Kind() == reflect.Ptr && val.Elem().Type().Implements(pluginIFace) {
					pluginInstance = val.Elem().Interface().(Plugin)
				}
			}
		}

		if pluginInstance != nil {
			if hh, ok := pluginInstance.(hooks.Handler); ok {
				hooks.Register(hh)
			}

			pluginInstance.RegisterCommands(rootCmd)
			return
		}
	}

	// Если не получилось получить Plugin интерфейс — пробуем функцию Describe + Execute
	descSym, derr := p.Lookup("Describe")
	if derr != nil {
		// нет Describe — ничего больше не делаем
		fmt.Printf("invalid Plugin type in %s\n", path)
		return
	}

	// Получаем JSON-описание команд из Describe (поддерживаем func() (string|[]byte) или переменную)
	var descBytes []byte
	dv := reflect.ValueOf(descSym)
	if dv.Kind() == reflect.Func {
		outs := dv.Call(nil)
		if len(outs) > 0 {
			r := outs[0]
			switch r.Kind() {
			case reflect.String:
				descBytes = []byte(r.String())
			case reflect.Slice:
				if r.Type().Elem().Kind() == reflect.Uint8 {
					descBytes = r.Bytes()
				}
			}
		}
	} else {
		// переменная Describe
		rv := dv
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.String {
			descBytes = []byte(rv.String())
		} else if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			descBytes = rv.Bytes()
		}
	}

	if len(descBytes) == 0 {
		fmt.Printf("invalid plugin description from %s: empty\n", path)
		return
	}

	var desc struct {
		Name     string `json:"name"`
		Commands []struct {
			Use   string `json:"use"`
			Short string `json:"short"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(descBytes, &desc); err != nil {
		fmt.Printf("invalid plugin description from %s: %v\n", path, err)
		return
	}

	// Найдём функцию-исполнитель в плагине
	var execSym interface{}
	for _, n := range []string{"Execute", "RunCommand", "HandleCommand"} {
		if s, e := p.Lookup(n); e == nil {
			execSym = s
			break
		}
	}
	if execSym == nil {
		fmt.Printf("plugin %s: no Execute/RunCommand/HandleCommand symbol\n", path)
		return
	}

	execVal := reflect.ValueOf(execSym)
	if execVal.Kind() != reflect.Func {
		// может быть указатель на функцию
		if execVal.Kind() == reflect.Ptr {
			execVal = execVal.Elem()
		}
		if execVal.Kind() != reflect.Func {
			fmt.Printf("plugin %s: execute symbol is not a function\n", path)
			return
		}
	}

	// helper: вызывает функцию-плагин с разными сигнатурами
	runExec := func(cmdName string, args []string) error {
		t := execVal.Type()
		var in []reflect.Value
		// поддерживаем сигнатуры: func(cmd string, args []string) (error|int), func(args []string) (error|int), func() (error|int)
		if t.NumIn() == 2 && t.In(0).Kind() == reflect.String && t.In(1).Kind() == reflect.Slice && t.In(1).Elem().Kind() == reflect.String {
			in = []reflect.Value{reflect.ValueOf(cmdName), reflect.ValueOf(args)}
		} else if t.NumIn() == 1 && t.In(0).Kind() == reflect.Slice && t.In(0).Elem().Kind() == reflect.String {
			in = []reflect.Value{reflect.ValueOf(args)}
		} else if t.NumIn() == 0 {
			in = nil
		} else {
			fmt.Printf("plugin %s: unsupported Execute signature\n", path)
			return nil
		}

		outs := execVal.Call(in)
		if len(outs) == 0 {
			return nil
		}
		// первый возвращаемый параметр может быть error или int
		r0 := outs[0]
		if r0.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !r0.IsNil() {
				return r0.Interface().(error)
			}
			return nil
		}
		if r0.Kind() == reflect.Int {
			if r0.Int() != 0 {
				return fmt.Errorf("plugin exit code %d", r0.Int())
			}
			return nil
		}
		return nil
	}

	for _, c := range desc.Commands {
		sub := &cobra.Command{
			Use:   c.Use,
			Short: c.Short,
			RunE: func(p string) func(cmd *cobra.Command, args []string) error {
				return func(cmd *cobra.Command, args []string) error {
					return runExec(p, args)
				}
			}(c.Use),
		}
		rootCmd.AddCommand(sub)
	}
}
