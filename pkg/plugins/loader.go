package plugins

import (
	"fmt"
	"plugin"
)

func LoadPlugin(path string, args []string) (Plugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %v", err)
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("failed to find symbol 'Plugin': %v", err)
	}

	pluginInstance, ok := sym.(Plugin)
	if !ok {
		return nil, fmt.Errorf("symbol 'Plugin' does not implement Plugin interface")
	}

	pluginInstance.SetArgs(args)
	return pluginInstance, nil
}
