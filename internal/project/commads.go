package project

import (
	"strings"

	"github.com/spf13/cobra"
)

func RegisterCommands(rootCmd *cobra.Command) {
	prjCmd := &cobra.Command{
		Use:   "project",
		Short: "Project related commands",
	}

	var (
		fromURL  string
		name     string
		dir      string
		gitInit  bool
		lang     string
		provider string
	)

	newCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project (template or Git URL)",
		Long: `Create a new project.

Examples:
  forge project create                  # interactive wizard: language / empty / Git repository
  forge project create --lang go        # generate a Go project immediately
  forge project create --from https://...  # clone from Git template
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			from := strings.TrimSpace(fromURL)
			l := strings.TrimSpace(lang)
			prov := TemplateProvider(strings.TrimSpace(provider))

			// 1) template / URL mode
			if from != "" {
				resolved, err := ResolveTemplateURL(from, prov)
				if err != nil {
					return err
				}

				projName := strings.TrimSpace(name)
				if projName == "" {
					projName = deriveProjectName(resolved)
					if projName == "" {
						projName = "forge-project"
					}
				}

				targetDir := strings.TrimSpace(dir)
				if targetDir == "" {
					targetDir = "./" + projName
				}

				return CreateProjectFromGit(resolved, projName, targetDir, gitInit)
			}

			// 2) lang mode
			if l != "" {
				projName := strings.TrimSpace(name)
				if projName == "" {
					projName = "forge-project"
				}
				targetDir := strings.TrimSpace(dir)
				if targetDir == "" {
					targetDir = "./" + projName
				}
				return InitProject(l, projName, targetDir, gitInit)
			}

			// 3) wizard mode
			return runProjectNewWizard(name, dir, gitInit)
		},
	}

	newCmd.Flags().StringVar(&fromURL, "from", "", "Git template (URL or owner/repo)")
	newCmd.Flags().StringVar(&provider, "provider", "github", "Template provider: github|gitlab|bitbucket")
	newCmd.Flags().StringVar(&lang, "lang", "", "Project language: go, node, ts, vue, empty")
	newCmd.Flags().StringVar(&name, "name", "", "Project name")
	newCmd.Flags().StringVar(&dir, "dir", "", "Target directory (default: ./<name>)")
	newCmd.Flags().BoolVar(&gitInit, "git-init", false, "Initialize git in created project")

	var (
		targetDir string
		remoteURL string
	)

	gitCmd := &cobra.Command{
		Use:   "git:add",
		Short: "Initialize Git and add a remote repository",
		Long: `Initialize Git repository (if missing), create initial commit, 
add remote and optionally push to it.

Examples:
  forge project git:add
  forge project git:add --remote git@github.com:user/project.git
  forge project git:add --dir ./services/api
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := strings.TrimSpace(targetDir)
			if dir == "" {
				dir = "."
			}

			remote := strings.TrimSpace(remoteURL)

			return GitAddProcess(dir, remote)
		},
	}

	gitCmd.Flags().StringVar(&targetDir, "dir", "", "Target project directory")
	gitCmd.Flags().StringVar(&remoteURL, "remote", "", "Remote Git URL")

	prjCmd.AddCommand(newCmd, gitCmd)
	rootCmd.AddCommand(prjCmd)
}
