package project

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

func runProjectNewWizard(nameFlag, dirFlag string, gitInit bool) error {
	mode, err := selectNewProjectMode()
	if err != nil {
		return err
	}

	var fromURL string
	var lang string

	switch mode {
	case "git":
		fromURL, err = askGitURL()
		if err != nil {
			return err
		}
	case "go", "node", "ts", "empty":
		lang = mode
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}

	suggestedName := ""
	if fromURL != "" {
		suggestedName = deriveProjectName(fromURL)
	}

	projectName, targetDir, err := askProjectNameAndDir(nameFlag, dirFlag, suggestedName)
	if err != nil {
		return err
	}

	if targetDir != "." {
		targetDir = filepath.Clean(targetDir)
	}

	if fromURL != "" {
		return CreateProjectFromGit(fromURL, projectName, targetDir, gitInit)
	}

	return InitProject(lang, projectName, targetDir, gitInit)
}

func selectNewProjectMode() (string, error) {
	prompt := &survey.Select{
		Message: "What do you want to create?",
		Options: []string{
			"Go project",
			"Node.js project",
			"TypeScript project",
			"Empty project",
			"From Git repository URL",
		},
		Default: "Go project",
	}

	var answer string
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}

	switch answer {
	case "Go project":
		return "go", nil
	case "Node.js project":
		return "node", nil
	case "TypeScript project":
		return "ts", nil
	case "Empty project":
		return "empty", nil
	case "From Git repository URL":
		return "git", nil
	default:
		return "", fmt.Errorf("unexpected choice: %s", answer)
	}
}

func askGitURL() (string, error) {
	prompt := &survey.Input{
		Message: "Enter Git repository URL:",
		Help:    "Example: https://github.com/user/template.git",
	}

	var url string
	if err := survey.AskOne(prompt, &url, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return strings.TrimSpace(url), nil
}

func askProjectNameAndDir(nameFlag, dirFlag, suggestedName string) (string, string, error) {
	// --- 1) Имя проекта ---
	defaultName := strings.TrimSpace(nameFlag)
	if defaultName == "" {
		if suggestedName != "" {
			defaultName = suggestedName
		} else {
			defaultName = "forge-project"
		}
	}

	namePrompt := &survey.Input{
		Message: "Project name:",
		Default: defaultName,
	}

	var projectName string
	if err := survey.AskOne(namePrompt, &projectName, survey.WithValidator(survey.Required)); err != nil {
		return "", "", err
	}
	projectName = strings.TrimSpace(projectName)

	// --- 2) Папка проекта ---
	defaultDir := strings.TrimSpace(dirFlag)
	if defaultDir == "" {
		defaultDir = "./" + projectName
	}

	dirPrompt := &survey.Input{
		Message: "Target directory ('.' = current directory):",
		Default: defaultDir,
	}

	var targetDir string
	if err := survey.AskOne(dirPrompt, &targetDir, survey.WithValidator(survey.Required)); err != nil {
		return "", "", err
	}
	targetDir = strings.TrimSpace(targetDir)

	return projectName, targetDir, nil
}
