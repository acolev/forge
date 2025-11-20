package main

import (
	"fmt"
	"forge/internal/migrations"
	"forge/internal/plugins"
	"forge/internal/project"
	"forge/internal/seeders"
	"forge/internal/selfupdate"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const Version = "2.0.6"

func main() {

	rootCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge CLI – Project & Dev Toolkit",
		Long: `Forge is a modern CLI toolkit for developers.

It provides:
  • Project scaffolding (Go, Node, TS, templates)
  • Database migrations (make, migrate, rollback)
  • Git integration for new/existing projects
  • Environment initialization and management
  • Self-update system
  • Plugin support

Use "forge --help" to see available commands.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Specify a command. Use 'forge --help' for more information.")
		},
		Version: Version,
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create or add environment variables to .env file",
		RunE: func(cmd *cobra.Command, args []string) error {
			envFilePath := ".env"
			dbSettings := []string{
				"DB_DRIVER=sqlite",
				"#DB_HOST=localhost",
				"#DB_PORT=5432",
				"#DB_USER=user",
				"#DB_PASSWORD=password",
				"#DB_NAME=database",
				"PLUGINS_DIR=database/plugins",
			}

			file, err := os.OpenFile(envFilePath, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return fmt.Errorf("unable to open or create .env file: %v", err)
			}
			defer file.Close()

			content, err := ioutil.ReadAll(file)
			if err != nil {
				return fmt.Errorf("unable to read .env file: %v", err)
			}

			for _, setting := range dbSettings {
				if !strings.Contains(string(content), setting) {
					if _, err := file.WriteString(setting + "\n"); err != nil {
						return fmt.Errorf("unable to write to .env file: %v", err)
					}
				}
			}

			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "env",
		Short: "Display the current framework environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := ioutil.ReadFile(".env")
			if err != nil {
				return fmt.Errorf("unable to read .env file: %v", err)
			}
			fmt.Println(string(content))
			return nil
		},
	})

	selfupdate.Register(rootCmd, Version)
	migrations.RegisterCommands(rootCmd)
	seeders.RegisterCommands(rootCmd)
	project.RegisterCommands(rootCmd)

	// Load plugins and register their commands
	plugins.LoadPlugins(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
