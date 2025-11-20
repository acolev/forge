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

	var (
		fromRef string
		fromURL string
		lang    string
	)

	switch mode {
	case "git":
		// Ask for template reference: can be a full URL or owner/repo.
		fromRef, err = askGitRef()
		if err != nil {
			return err
		}

		// Resolve "owner/repo" into a full URL using GitHub as default provider.
		fromURL, err = ResolveTemplateURL(fromRef, ProviderGitHub)
		if err != nil {
			return err
		}

	case "go", "node", "ts", "empty":
		lang = mode

	default:
		return fmt.Errorf("unknown project mode: %s", mode)
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
			"From Git repository or template",
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
	case "From Git repository or template":
		return "git", nil
	default:
		return "", fmt.Errorf("unexpected choice: %s", answer)
	}
}

func askGitRef() (string, error) {
	prompt := &survey.Input{
		Message: "Enter Git repository (URL or owner/repo):",
		Help:    "Examples: https://github.com/acolev/forge.git or acolev/forge",
	}

	var ref string
	if err := survey.AskOne(prompt, &ref, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return strings.TrimSpace(ref), nil
}

func askProjectNameAndDir(nameFlag, dirFlag, suggestedName string) (string, string, error) {
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
