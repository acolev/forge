package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	githubOwner = "acolev"
	githubRepo  = "forge"

	// Naming pattern for your release assets.
	// Example: forge-darwin-amd64, forge-linux-arm64, etc.
	assetPattern = "forge-%s-%s" // GOOS-GOARCH
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// RunSelfUpdate checks GitHub releases and replaces the current binary.
func RunSelfUpdate(currentVersion string) error {
	fmt.Println("Checking for updates...")

	latest, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	cur := strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")

	if cur != "" && cur == latestVersion {
		fmt.Printf("You are already on the latest version (%s).\n", latestVersion)
		return nil
	}

	if cur == "" {
		fmt.Printf("Current version is unknown, latest is %s.\n", latestVersion)
	} else {
		fmt.Printf("Current version: %s, latest: %s.\n", cur, latestVersion)
	}

	assetURL, assetName, err := selectAsset(latest)
	if err != nil {
		return err
	}

	fmt.Printf("Selected asset: %s\n", assetName)
	fmt.Println("Downloading binary...")

	tmpFile, err := downloadToTemp(assetURL)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := makeExecutable(tmpFile); err != nil {
		return fmt.Errorf("failed to make downloaded file executable: %w", err)
	}

	if err := replaceCurrentBinary(tmpFile); err != nil {
		return fmt.Errorf("failed to replace current binary: %w", err)
	}

	fmt.Println("Update complete!")
	return nil
}

// fetchLatestRelease retrieves the most recent GitHub release metadata.
func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API error: %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// selectAsset finds the most appropriate release asset for the current OS/arch.
func selectAsset(rel *githubRelease) (downloadURL, name string, err error) {
	if len(rel.Assets) == 0 {
		return "", "", errors.New("no assets found in the latest release")
	}

	expected := fmt.Sprintf(assetPattern, runtime.GOOS, runtime.GOARCH)
	for _, a := range rel.Assets {
		if a.Name == expected {
			return a.BrowserDownloadURL, a.Name, nil
		}
	}

	// Fallback: return the first available asset
	// You should adjust assetPattern to properly match your release files.
	if len(rel.Assets) > 0 {
		a := rel.Assets[0]
		return a.BrowserDownloadURL, a.Name, fmt.Errorf(
			"no matching asset for %s/%s, using fallback asset %s; update assetPattern if needed",
			runtime.GOOS, runtime.GOARCH, a.Name,
		)
	}

	return "", "", errors.New("no suitable asset found")
}

// downloadToTemp downloads the asset binary to a temporary file.
func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "forge-selfupdate-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// makeExecutable adds +x permission to the file.
func makeExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.Chmod(path, info.Mode()|0o111)
}

// replaceCurrentBinary atomically replaces the running binary with the new binary.
func replaceCurrentBinary(newBinaryPath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve symlinks for current executable: %w", err)
	}

	fmt.Printf("Replacing %s with %s\n", exePath, newBinaryPath)

	return os.Rename(newBinaryPath, exePath)
}
