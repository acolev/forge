package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"forge/internal/config"

	"github.com/spf13/cobra"
)

var pluginModuleNameSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

const availableHooksHelp = `Available hooks:
  - db.migrate.before
  - db.migrate.after
  - project.create.before
  - project.create.after

Hook notes:
  - Hook payloads are not forwarded to plugins yet.
  - Go source plugins can be declared with "lang": "go" and "source": "src".
  - Forge builds source-based Go plugins automatically before execution.
`

func RegisterManagementCommands(rootCmd *cobra.Command, projectDir string) {
	var createGlobal bool
	var buildGlobal bool
	var createHook string

	pluginsCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage Forge plugins",
		Long: `Manage local and global Forge plugins.

Local plugins live in .forge/plugins inside the current project.
Global plugins live in ~/.forge/plugins.

Supported lifecycle:
  forge plugins create <vendor>/<name>
  forge plugins build <vendor>/<name>
  forge plugins install <vendor>/<name>

` + availableHooksHelp,
	}

	createCmd := &cobra.Command{
		Use:   "create <vendor>/<name>",
		Short: "Create a new plugin scaffold",
		Long: `Create a new plugin scaffold.

By default the plugin is created locally inside .forge/plugins.
Use --global to scaffold directly into ~/.forge/plugins.

Examples:
  forge plugins create bookly/migrate
  forge plugins create bookly/migrate --hook db.migrate.before
  forge plugins create bookly/migrate --global

If --hook is provided, Forge generates a hook-oriented manifest and Go source
template for that event.

` + availableHooksHelp,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vendor, name, err := parsePluginSlug(args[0])
			if err != nil {
				return err
			}
			rootDir, err := pluginRootDir(projectDir, createGlobal)
			if err != nil {
				return err
			}
			pluginDir, err := CreatePluginScaffold(rootDir, vendor, name, strings.TrimSpace(createHook))
			if err != nil {
				return err
			}
			fmt.Printf("Created plugin scaffold at %s\n", pluginDir)
			return nil
		},
	}
	createCmd.Flags().BoolVar(&createGlobal, "global", false, "Create the plugin in the global plugin directory")
	createCmd.Flags().StringVar(&createHook, "hook", "", "Create a hook-oriented scaffold, for example db.migrate.before")

	buildCmd := &cobra.Command{
		Use:   "build <vendor>/<name>",
		Short: "Build a Go plugin from source",
		Long: `Build a source-based Go plugin.

By default Forge builds the local plugin from .forge/plugins.
Use --global to build from ~/.forge/plugins.

Example:
  forge plugins build bookly/migrate
  forge plugins build bookly/migrate --global
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolvePluginBySlug(projectDir, args[0], buildGlobal)
			if err != nil {
				return err
			}
			outPath, err := BuildPlugin(p)
			if err != nil {
				return err
			}
			fmt.Printf("Built plugin binary: %s\n", outPath)
			return nil
		},
	}
	buildCmd.Flags().BoolVar(&buildGlobal, "global", false, "Build from the global plugin directory")

	installCmd := &cobra.Command{
		Use:   "install <vendor>/<name>",
		Short: "Install a local plugin into the global plugin directory",
		Long: `Install a local plugin into the global plugin directory.

This copies .forge/plugins/<vendor>/<name> to ~/.forge/plugins/<vendor>/<name>.

Example:
  forge plugins install bookly/migrate
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolvePluginBySlug(projectDir, args[0], false)
			if err != nil {
				return err
			}
			destDir, err := InstallPluginGlobally(p)
			if err != nil {
				return err
			}
			fmt.Printf("Installed plugin globally: %s\n", destDir)
			return nil
		},
	}

	pluginsCmd.AddCommand(createCmd, buildCmd, installCmd)
	rootCmd.AddCommand(pluginsCmd)
}

func CreatePluginScaffold(rootDir, vendor, name, hookName string) (string, error) {
	pluginDir := filepath.Join(rootDir, vendor, name)
	if _, err := os.Stat(pluginDir); err == nil {
		return "", fmt.Errorf("plugin already exists: %s", pluginDir)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat plugin dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		return "", fmt.Errorf("create plugin dir: %w", err)
	}

	manifest := PluginManifest{
		Name:        name,
		Vendor:      vendor,
		Namespace:   vendor + "-" + name,
		Description: fmt.Sprintf("%s/%s plugin", vendor, name),
		Lang:        "go",
		Entry:       name,
		Source:      "src",
		Commands: []PluginCommand{
			{
				Name:        "ping",
				Description: "Ping the plugin runtime",
			},
		},
		Hooks: map[string]HookConfig{},
	}
	if hookName != "" {
		manifest.Commands = nil
		manifest.Hooks[hookName] = HookConfig{
			Command: hookCommandName(hookName),
		}
	}

	manifestPath := filepath.Join(pluginDir, "plugin.json")
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		return "", err
	}

	goMod := fmt.Sprintf("module %s\n\ngo 1.26.0\n", pluginModuleName(vendor, name))
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "go.mod"), []byte(goMod), 0o644); err != nil {
		return "", fmt.Errorf("write go.mod: %w", err)
	}

	mainGo := pluginMainTemplate(vendor, name, hookName)

	if err := os.WriteFile(filepath.Join(pluginDir, "src", "main.go"), []byte(mainGo), 0o644); err != nil {
		return "", fmt.Errorf("write main.go: %w", err)
	}

	return pluginDir, nil
}

func pluginMainTemplate(vendor, name, hookName string) string {
	if hookName == "db.migrate.before" {
		return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type pluginEvent struct {
	Name string `+"`json:\"name\"`"+`
}

type pluginRequest struct {
	Type       string       `+"`json:\"type\"`"+`
	Command    string       `+"`json:\"command\"`"+`
	ProjectDir string       `+"`json:\"project_dir\"`"+`
	Event      *pluginEvent `+"`json:\"event\"`"+`
}

type pluginResponse struct {
	OK      bool     `+"`json:\"ok\"`"+`
	Message string   `+"`json:\"message\"`"+`
	Logs    []string `+"`json:\"logs\"`"+`
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeResponse(false, "read stdin: "+err.Error(), nil)
		return
	}

	var req pluginRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeResponse(false, "decode request: "+err.Error(), nil)
		return
	}

	logs := []string{"[@%s/%s] db.migrate.before hook triggered"}
	if req.ProjectDir != "" {
		logs = append(logs, fmt.Sprintf("[@%s/%s] project_dir=%%s", req.ProjectDir))
	}

	if req.Event != nil && req.Event.Name == "db.migrate.before" {
		// TODO: initialize GORM here and call AutoMigrate(...) before SQL migrations run.
		writeResponse(true, "[@%s/%s] before migration hook executed", logs)
		return
	}

	writeResponse(true, "[@%s/%s] plugin executed", logs)
}

func writeResponse(ok bool, message string, logs []string) {
	resp := pluginResponse{OK: ok, Message: message, Logs: logs}
	encoded, _ := json.Marshal(resp)
	fmt.Print(string(encoded))
}
	`, vendor, name, vendor, name, vendor, name, vendor, name)
	}

	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type pluginEvent struct {
	Name string `+"`json:\"name\"`"+`
}

type pluginRequest struct {
	Type       string       `+"`json:\"type\"`"+`
	Command    string       `+"`json:\"command\"`"+`
	ProjectDir string       `+"`json:\"project_dir\"`"+`
	Event      *pluginEvent `+"`json:\"event\"`"+`
}

type pluginResponse struct {
	OK      bool     `+"`json:\"ok\"`"+`
	Message string   `+"`json:\"message\"`"+`
	Logs    []string `+"`json:\"logs\"`"+`
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeResponse(false, "read stdin: "+err.Error(), nil)
		return
	}

	var req pluginRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeResponse(false, "decode request: "+err.Error(), nil)
		return
	}

	logs := []string{fmt.Sprintf("[@%s/%s] request type=%%s command=%%s", req.Type, req.Command)}
	if req.Event != nil && req.Event.Name != "" {
		logs = append(logs, fmt.Sprintf("[@%s/%s] event=%%s", req.Event.Name))
	}

	writeResponse(true, "[@%s/%s] plugin executed", logs)
}

func writeResponse(ok bool, message string, logs []string) {
	resp := pluginResponse{OK: ok, Message: message, Logs: logs}
	encoded, _ := json.Marshal(resp)
	fmt.Print(string(encoded))
}
`, vendor, name, vendor, name, vendor, name)
}

func hookCommandName(hookName string) string {
	replacer := strings.NewReplacer(".", "-", "_", "-")
	return replacer.Replace(hookName)
}

func InstallPluginGlobally(p Plugin) (string, error) {
	globalRoot, err := globalPluginRootDir()
	if err != nil {
		return "", err
	}

	destDir := filepath.Join(globalRoot, p.Manifest.Vendor, p.Manifest.Name)
	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("remove previous plugin install: %w", err)
	}
	if err := copyDir(p.BaseDir, destDir); err != nil {
		return "", err
	}
	return destDir, nil
}

func resolvePluginBySlug(projectDir, slug string, useGlobal bool) (Plugin, error) {
	vendor, name, err := parsePluginSlug(slug)
	if err != nil {
		return Plugin{}, err
	}

	rootDir, err := pluginRootDir(projectDir, useGlobal)
	if err != nil {
		return Plugin{}, err
	}

	return loadPluginFromManifest(filepath.Join(rootDir, vendor, name, "plugin.json"))
}

func pluginRootDir(projectDir string, useGlobal bool) (string, error) {
	if useGlobal {
		return globalPluginRootDir()
	}
	return config.ResolvePluginsDir(projectDir)
}

func globalPluginRootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".forge", "plugins"), nil
}

func parsePluginSlug(slug string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(slug), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid plugin slug %q, expected vendor/name", slug)
	}
	return parts[0], parts[1], nil
}

func pluginModuleName(vendor, name string) string {
	raw := strings.ToLower(vendor + "-" + name + "-plugin")
	return pluginModuleNameSanitizer.ReplaceAllString(raw, "-")
}

func writeJSONFile(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
