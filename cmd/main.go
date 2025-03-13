package main

import (
	"errors"
	"fmt"
	"forge/pkg/database"
	"forge/pkg/migrations"
	"forge/pkg/plugins"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func main() {

	rootCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge CLI - Database Migrations Manager",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Specify a command. Use 'forge --help' for more information.")
		},
		Version: "1.3.0",
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
		Use:   "make:migration",
		Short: "Create a new database migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("you must specify a table name")
			}
			tableName := args[0]
			return migrations.CreateMigration(tableName)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, _ := database.InitDB()
			return migrations.RunMigrations(db)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "migrate:rollback",
		Short: "Rollback the last database migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, _ := database.InitDB()
			return migrations.RollbackLastMigration(db)
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

	rootCmd.AddCommand(&cobra.Command{
		Use:   "plug",
		Short: "Load and execute a plugin based on the filename",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("you must specify the plugin filename")
			}
			pluginFilename := args[0]

			pluginDir := "./"

			if _, err := os.Stat(".env"); err == nil {
				content, err := ioutil.ReadFile(".env")
				if err != nil {
					return fmt.Errorf("unable to read .env file: %v", err)
				}

				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "PLUGINS_DIR=") {
						pluginDir = strings.TrimPrefix(line, "PLUGINS_DIR=")
						break
					}
				}
			}

			pluginPath := fmt.Sprintf("%s/%s.so", pluginDir, pluginFilename)
			pluginArgs := args[1:]
			pluginInstance, err := plugins.LoadPlugin(pluginPath, pluginArgs)
			if err != nil {
				return err
			}
			fmt.Printf("Loaded plugin: %s\n", pluginInstance.Name())
			return pluginInstance.Execute()
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
