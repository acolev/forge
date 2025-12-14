package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Loader struct {
	projectDir string
}

func NewLoader(projectDir string) *Loader {
	return &Loader{projectDir: projectDir}
}

// ScanPlugins ищет plugin.json в:
func (l *Loader) ScanPlugins() ([]Plugin, error) {
	var plugins []Plugin

	// локальные плагины
	localRoot := filepath.Join(l.projectDir, ".forge", "plugins")
	_ = filepath.Walk(localRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() != "plugin.json" {
			return nil
		}
		if p, err := loadPluginFromManifest(path); err == nil {
			plugins = append(plugins, p)
		} else {
			fmt.Println("[forge][plugin] failed to load", path, ":", err)
		}
		return nil
	})

	// глобальные плагины
	if home, err := os.UserHomeDir(); err == nil {
		globalRoot := filepath.Join(home, ".forge", "plugins")
		_ = filepath.Walk(globalRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if info.Name() != "plugin.json" {
				return nil
			}
			if p, err := loadPluginFromManifest(path); err == nil {
				plugins = append(plugins, p)
			} else {
				fmt.Println("[forge][plugin] failed to load", path, ":", err)
			}
			return nil
		})
	}

	return plugins, nil
}

func loadPluginFromManifest(manifestPath string) (Plugin, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Plugin{}, err
	}

	var mf PluginManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return Plugin{}, err
	}

	if mf.Name == "" || mf.Namespace == "" || mf.Entry == "" {
		return Plugin{}, fmt.Errorf("invalid manifest (name/namespace/entry required)")
	}

	baseDir := filepath.Dir(manifestPath)

	return Plugin{
		Manifest: mf,
		BaseDir:  baseDir,
	}, nil
}
