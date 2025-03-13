package migrations

import (
	"errors"
	"forge/pkg/database"
	"github.com/spf13/cobra"
)

func RegisterCommands(rootCmd *cobra.Command) {

	rootCmd.AddCommand(&cobra.Command{
		Use:   "make:migration",
		Short: "Create a new database migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("you must specify a table name")
			}
			tableName := args[0]
			return CreateMigration(tableName)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, _ := database.InitDB()
			return RunMigrations(db)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "migrate:rollback",
		Short: "Rollback the last database migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, _ := database.InitDB()
			return RollbackLastMigration(db)
		},
	})

}
