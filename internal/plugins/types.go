package plugins

import "path/filepath"

type PluginManifest struct {
	Name        string                `json:"name"`
	Vendor      string                `json:"vendor"`
	Namespace   string                `json:"namespace"`
	Description string                `json:"description"`
	Lang        string                `json:"lang"`  // runtime: binary|node|php...
	Entry       string                `json:"entry"` // файл/бинарь для запуска
	Commands    []PluginCommand       `json:"commands"`
	Hooks       map[string]HookConfig `json:"hooks"`
}

type PluginCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type HookConfig struct {
	Command string `json:"command"` // имя команды внутри плагина, которую надо вызвать
}

// Загруженный плагин
type Plugin struct {
	Manifest PluginManifest
	BaseDir  string // путь к .forge/plugins/vendor/plugin_name
}

func (p Plugin) EntryPath() string {
	return filepath.Join(p.BaseDir, p.Manifest.Entry)
}

// ----------------- Протокол forge <-> плагин -------------------

type RequestType string

const (
	RequestTypeCommand RequestType = "command"
	RequestTypeEvent   RequestType = "event"
)

type PluginEvent struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload,omitempty"`
}

type PluginRequest struct {
	Type       RequestType       `json:"type"` // "command" или "event"
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	ProjectDir string            `json:"project_dir"`
	Event      *PluginEvent      `json:"event,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

type PluginResponse struct {
	OK      bool     `json:"ok"`
	Message string   `json:"message"`
	Logs    []string `json:"logs"`
}
