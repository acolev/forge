package main

import (
	"errors"
	"fmt"
	"forge/pkg/migrations"
	"log"
	"os"

	"github.com/joho/godotenv"
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

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

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
			return migrations.RunMigrations()
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
