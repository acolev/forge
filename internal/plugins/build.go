package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func BuildPlugin(p Plugin) (string, error) {
	if p.Manifest.Lang != "go" || p.Manifest.Source == "" {
		return "", fmt.Errorf("plugin %s does not declare Go source", p.Manifest.Name)
	}

	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("go toolchain is required to build plugin %s: %w", p.Manifest.Name, err)
	}

	outPath := p.EntryPath()
	cmd := exec.Command("go", "build", "-o", outPath, ".")
	cmd.Dir = p.SourcePath()
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build plugin %s: %w\n%s", p.Manifest.Name, err, string(output))
	}
	return outPath, nil
}

func EnsurePluginExecutable(p Plugin) error {
	if p.Manifest.Lang != "go" || p.Manifest.Source == "" {
		return nil
	}

	entryInfo, entryErr := os.Stat(p.EntryPath())
	sourceModTime, sourceErr := latestSourceModTime(p.SourcePath())
	if sourceErr != nil {
		return fmt.Errorf("stat plugin source %s: %w", p.SourcePath(), sourceErr)
	}

	if entryErr == nil && !sourceModTime.After(entryInfo.ModTime()) {
		return nil
	}

	_, err := BuildPlugin(p)
	return err
}

func latestSourceModTime(root string) (time.Time, error) {
	var latest time.Time

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return latest, nil
}
