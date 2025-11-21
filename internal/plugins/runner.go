package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func buildExecCommand(p Plugin) (string, []string) {
	entryPath := p.EntryPath()

	switch p.Manifest.Lang {
	case "node":
		return "node", []string{entryPath}
	case "python":
		return "python", []string{entryPath}
	case "binary", "":
		// по умолчанию считаем, что entry — бинарник
		return entryPath, nil
	default:
		// можно расширить для других рантаймов
		return entryPath, nil
	}
}

func RunPlugin(p Plugin, req PluginRequest) (*PluginResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	cmdName, cmdArgs := buildExecCommand(p)
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Dir = p.BaseDir

	cmd.Stdin = bytes.NewReader(payload)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "[forge][plugin]", p.Manifest.Name, "stderr:", stderr.String())
		return nil, fmt.Errorf("plugin %s execution error: %w", p.Manifest.Name, err)
	}

	var resp PluginResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("plugin %s: invalid JSON response: %w", p.Manifest.Name, err)
	}

	return &resp, nil
}
