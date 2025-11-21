package plugins

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"strings"
)

func LoadPlugins(rootCmd *cobra.Command) {
	pluginDir := "database/plugins"

	if _, err := os.Stat(".env"); err == nil {
		content, err := ioutil.ReadFile(".env")
		if err != nil {
			fmt.Printf("unable to read .env file: %v\n", err)
			return
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PLUGINS_DIR=") {
				pluginDir = strings.TrimPrefix(line, "PLUGINS_DIR=")
				break
			}
		}
	}

	files, err := ioutil.ReadDir(pluginDir)
	if err != nil {
		fmt.Printf("unable to read plugin directory: %v\n", err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".so" {
			pluginPath := filepath.Join(pluginDir, file.Name())
			p, err := plugin.Open(pluginPath)
			if err != nil {
				fmt.Printf("unable to open plugin %s: %v\n", pluginPath, err)
				continue
			}

			sym, err := p.Lookup("Plugin")
			if err != nil {
				fmt.Printf("unable to find Plugin symbol in %s: %v\n", pluginPath, err)
				continue
			}

			pluginInstance, ok := sym.(Plugin)
			if !ok {
				fmt.Printf("invalid Plugin type in %s\n", pluginPath)
				continue
			}

			pluginInstance.RegisterCommands(rootCmd)
			fmt.Printf("Loaded plugin: %s\n", pluginInstance.Name())
		}
	}
}
