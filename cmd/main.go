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

const Version = "2.1.1"

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
  • Upgrade system
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

			content, err := os.ReadFile(envFilePath)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("unable to read %s: %v", envFilePath, err)
				}
				content = nil
			}

			file, err := os.OpenFile(envFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				return fmt.Errorf("unable to open or create %s: %v", envFilePath, err)
			}
			defer file.Close()

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
				if os.IsNotExist(err) {
					fmt.Printf("# %s\n", settings.EnvFile)
					fmt.Printf("%s=%s\n", config.ForgeDBDSNKey, settings.DBDSN)
					fmt.Printf("%s=%s\n", config.ForgePluginsDirKey, settings.PluginsDir)
					return nil
				}
				return fmt.Errorf("unable to read %s: %v", settings.EnvFile, err)
			}
			fmt.Printf("# %s\n", settings.EnvFile)
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
		projectDir = "."
	}

	plugins.RegisterManagementCommands(rootCmd, projectDir)

	pm, err := plugins.NewManager(projectDir)
	if err != nil {
		log.Println("[forge][plugins] load error:", err)
	} else {
		pm.RegisterCommands(rootCmd)
		hooks.Register(plugins.NewHookHandler(pm))
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
