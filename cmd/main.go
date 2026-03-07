package main

import (
	"fmt"
	"forge/internal/config"
	"forge/internal/hooks"
	"forge/internal/migrations"
	"forge/internal/plugins"
	"forge/internal/project"
	"forge/internal/seeders"
	"forge/internal/selfupdate"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const Version = "2.0.9"

func main() {

	rootCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge CLI – Project & Dev Toolkit",
		Long: `
 ███████╗ ██████╗ ██████╗  ██████╗ ███████╗
 ██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
 █████╗  ██║   ██║██████╔╝██║  ███╗█████╗  
 ██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝  
 ██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
 ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝

Forge is a modern CLI toolkit for developers.

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
		Short: "Create or update .env.forge with Forge settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			envFilePath := config.DefaultEnvFile
			dbSettings := config.DefaultEnvLines()

			file, err := os.OpenFile(envFilePath, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return fmt.Errorf("unable to open or create %s: %v", envFilePath, err)
			}
			defer file.Close()

			content, err := os.ReadFile(envFilePath)
			if err != nil {
				return fmt.Errorf("unable to read %s: %v", envFilePath, err)
			}

			for _, setting := range dbSettings {
				if !strings.Contains(string(content), setting) {
					if _, err := file.WriteString(setting + "\n"); err != nil {
						return fmt.Errorf("unable to write to %s: %v", envFilePath, err)
					}
				}
			}

			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "env",
		Short: "Display Forge config environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := config.CurrentSettings()
			if err != nil {
				return err
			}

			content, err := os.ReadFile(settings.EnvFile)
			if err != nil {
				return fmt.Errorf("unable to read %s: %v", settings.EnvFile, err)
			}
			fmt.Println(string(content))
			return nil
		},
	})

	selfupdate.Register(rootCmd, Version)
	migrations.RegisterCommands(rootCmd)
	seeders.RegisterCommands(rootCmd)
	project.RegisterCommands(rootCmd)

	projectDir, err := os.Getwd()
	if err != nil {
		log.Println("[forge] cannot determine project dir:", err)
	} else {
		pm, err := plugins.NewManager(projectDir)
		if err != nil {
			log.Println("[forge][plugins] load error:", err)
		} else {
			pm.RegisterCommands(rootCmd)
			hooks.Register(plugins.NewHookHandler(pm))
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
