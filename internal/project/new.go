package project

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CreateProjectFromGit(repoURL, name, targetDir string, gitInit bool) error {
	fmt.Printf("Creating project %q from %s\n", name, repoURL)

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	if err := ensureTargetDirAvailable(targetDir); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "forge-template-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneDir := filepath.Join(tmpDir, "repo")

	fmt.Printf("Cloning template into %s...\n", cloneDir)
	if err := runCmd(tmpDir, "git", "clone", "--depth", "1", repoURL, cloneDir); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	if err := os.RemoveAll(filepath.Join(cloneDir, ".git")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove .git from template: %w", err)
	}

	if targetDir != "." {
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("failed to create target dir %s: %w", targetDir, err)
		}
	}

	if err := copyDir(cloneDir, targetDir); err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	fmt.Printf("Project created at %s\n", targetDir)

	if gitInit {
		if err := initGitRepo(targetDir); err != nil {
			return fmt.Errorf("project created, but git init failed: %w", err)
		}
	}

	return nil
}

func deriveProjectName(repoURL string) string {
	noQuery := strings.Split(repoURL, "?")[0]
	noHash := strings.Split(noQuery, "#")[0]

	base := filepath.Base(noHash)
	base = strings.TrimSuffix(base, ".git")
	base = strings.TrimSpace(base)

	if base == "" || base == "." || base == "/" {
		return ""
	}
	return base
}

func ensureTargetDirAvailable(targetDir string) error {
	if targetDir == "." {
		info, err := os.Stat(".")
		if err != nil {
			return fmt.Errorf("failed to stat current directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("current path is not a directory")
		}
		return nil
	}

	info, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		return nil // ок, создадим потом
	}
	if err != nil {
		return fmt.Errorf("failed to stat target dir: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path %s exists and is not a directory", targetDir)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("failed to read target dir: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("target directory %s is not empty", targetDir)
	}

	return nil
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
		if rel == "." {
			return nil
		}

		targetPath := dst
		if dst != "." {
			targetPath = filepath.Join(dst, rel)
		} else {
			targetPath = rel
		}

		if info.IsDir() {
			if targetPath != "." {
				if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
					return err
				}
			}
			return nil
		}

		return copyFile(path, targetPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

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

func initGitRepo(dir string) error {
	fmt.Println("Initializing new git repository...")

	steps := [][]string{
		{"git", "init"},
		{"git", "add", "."},
		{"git", "commit", "-m", "Initial commit (forge project)"},
	}

	for _, s := range steps {
		if err := runCmd(dir, s[0], s[1:]...); err != nil {
			return err
		}
	}

	return nil
}
