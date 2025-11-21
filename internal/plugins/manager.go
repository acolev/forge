package plugins

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// ---------------- Manager ----------------

type Manager struct {
	projectDir string
	plugins    []Plugin
}

func NewManager(projectDir string) (*Manager, error) {
	ldr := NewLoader(projectDir)
	plugs, err := ldr.ScanPlugins()
	if err != nil {
		return nil, err
	}
	return &Manager{
		projectDir: projectDir,
		plugins:    plugs,
	}, nil
}

// RegisterCommands навешивает команды вида: forge <namespace> <command>
func (m *Manager) RegisterCommands(root *cobra.Command) {
	nsMap := make(map[string]*cobra.Command)

	for _, p := range m.plugins {
		ns := p.Manifest.Namespace

		nsCmd, ok := nsMap[ns]
		if !ok {
			nsCmd = &cobra.Command{
				Use:   ns,
				Short: p.Manifest.Description,
			}
			nsMap[ns] = nsCmd
			root.AddCommand(nsCmd)
		}

		for _, pc := range p.Manifest.Commands {
			pluginCopy := p
			cmdName := pc.Name

			nsCmd.AddCommand(&cobra.Command{
				Use:   cmdName,
				Short: pc.Description,
				RunE: func(cmd *cobra.Command, args []string) error {
					req := PluginRequest{
						Type:       RequestTypeCommand,
						Command:    cmdName,
						Args:       args,
						ProjectDir: m.projectDir,
						Env: map[string]string{
							"FORGE_VERSION": "dev", // сюда потом подставишь реальную версию
						},
					}

					resp, err := RunPlugin(pluginCopy, req)
					if err != nil {
						return err
					}

					for _, l := range resp.Logs {
						fmt.Println("[plugin]", l)
					}
					if resp.Message != "" {
						fmt.Println(resp.Message)
					}
					if !resp.OK {
						return fmt.Errorf("plugin %s failed: %s", pluginCopy.Manifest.Name, resp.Message)
					}
					return nil
				},
			})
		}
	}
}

// ---------------- HookHandler ----------------

// HookHandler реализует hooks.Handler и бриджит события -> плагины
type HookHandler struct {
	m *Manager
}

func NewHookHandler(m *Manager) *HookHandler {
	return &HookHandler{m: m}
}

// Handle вызывается при hooks.Emit(ctx, hooks.Event{...})
func (h *HookHandler) Handle(ctx context.Context, name string, payload *any) error {
	for _, p := range h.m.plugins {
		hcfg, ok := p.Manifest.Hooks[name]
		if !ok {
			continue // плагин не подписан на это событие
		}

		// ВАЖНО: Payload сейчас вообще не тащим,
		// чтобы не ловить json: unsupported type (func, chan и т.д.)
		req := PluginRequest{
			Type:       RequestTypeEvent,
			Command:    hcfg.Command,
			ProjectDir: h.m.projectDir,
			Event: &PluginEvent{
				Name:    name,
				Payload: nil,
			},
		}

		resp, err := RunPlugin(p, req)
		if err != nil {
			return fmt.Errorf("plugin %s event %s error: %w", p.Manifest.Name, name, err)
		}

		for _, l := range resp.Logs {
			fmt.Println("[plugin]", l)
		}
		if resp.Message != "" {
			fmt.Println(resp.Message)
		}
		if !resp.OK {
			return fmt.Errorf("plugin %s event %s failed: %s", p.Manifest.Name, name, resp.Message)
		}
	}
	return nil
}
