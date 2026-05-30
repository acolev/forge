package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// UpdateEnvFile writes the given key/value pairs into the env file, replacing
// existing keys in place and appending new ones. Comments, blank lines and
// unrelated keys are preserved.
func UpdateEnvFile(path string, kv map[string]string) error {
	var lines []string
	if b, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(b), "\n")
		// Drop a single trailing empty line from the split so we don't grow blank lines.
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	seen := map[string]bool{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		if val, ok := kv[key]; ok {
			lines[i] = key + "=" + val
			seen[key] = true
		}
	}

	// Append keys not already present, in stable order.
	var missing []string
	for k := range kv {
		if !seen[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	for _, k := range missing {
		lines = append(lines, k+"="+kv[k])
	}

	out := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// ReadEnvFileValue reads a single key directly from an env file without
// touching the process environment (used for wizard pre-fill).
func ReadEnvFileValue(path, key string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			continue
		}
		if strings.TrimSpace(trimmed[:eq]) == key {
			return strings.TrimSpace(trimmed[eq+1:])
		}
	}
	return ""
}
