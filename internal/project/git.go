package project

import (
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GitAddProcess(dir, remote string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	fmt.Println("Using directory:", absDir)

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not found in PATH")
	}

	hasGit := dirExists(filepath.Join(absDir, ".git"))

	if !hasGit {
		fmt.Println("Git repository not found → initializing...")

		if err := runGit(absDir, "init"); err != nil {
			return err
		}
	}

	fmt.Println("Adding all files...")
	if err := runGit(absDir, "add", "."); err != nil {
		return err
	}

	if !hasGit || !gitHasCommits(absDir) {
		fmt.Println("Creating initial commit...")
		if err := runGit(absDir, "commit", "-m", "Initial commit (forge)"); err != nil {
			return err
		}
	}

	if remote == "" {
		remote, err = askRemoteURL()
		if err != nil {
			return err
		}
	}

	if gitHasOrigin(absDir) {
		return errors.New("origin remote already exists")
	}

	fmt.Println("Adding remote:", remote)
	if err := runGit(absDir, "remote", "add", "origin", remote); err != nil {
		return err
	}

	shouldPush, err := askYesNo("Push initial commit to remote?")
	if err != nil {
		return err
	}

	if shouldPush {
		fmt.Println("Pushing...")
		if err := runGit(absDir, "push", "-u", "origin", "main"); err != nil {
			return err
		}
	}

	fmt.Println("Git setup complete.")
	return nil
}

// ----------------------------------------------------
//                        HELPERS
// ----------------------------------------------------

func askRemoteURL() (string, error) {
	prompt := &survey.Input{
		Message: "Enter Git remote URL:",
		Help:    "Example: https://github.com/user/repo.git or git@github.com:user/repo.git",
	}

	var url string
	if err := survey.AskOne(prompt, &url, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return strings.TrimSpace(url), nil
}

func askYesNo(msg string) (bool, error) {
	prompt := &survey.Confirm{
		Message: msg,
		Default: true,
	}
	var answer bool
	err := survey.AskOne(prompt, &answer)
	return answer, err
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitHasCommits(dir string) bool {
	err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Run()
	return err == nil
}

func gitHasOrigin(dir string) bool {
	err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Run()
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
