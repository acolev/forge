package migrations

import (
	"bufio"
	"errors"
	"fmt"
	"forge/internal/database"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func RegisterCommands(rootCmd *cobra.Command) {

	migCmd := &cobra.Command{Use: "db", Short: "Database migration commands"}

	sqlCmd := &cobra.Command{
		Use:   "make:sql [table_name]",
		Short: "Create a SQL migration file",
		Long: `Create a new SQL migration.

If the name starts with a stub prefix, Forge will use a built-in or user stub.

Available stub templates:
  - create_table
  - update_table
  - add_column
  - drop_column
  - create_pivot_table
  - add_index
  - drop_index

Examples:
  forge make:sql create_table_users
  forge make:sql add_column_users_email
  forge make:sql
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var tableName string

			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				tableName = strings.TrimSpace(args[0])
			} else {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter table name: ")
				name, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read table name: %w", err)
				}
				tableName = strings.TrimSpace(name)
				if tableName == "" {
					return errors.New("table name cannot be empty")
				}
			}

			return CreateMigration(tableName)
		},
	}

	upCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			return RunMigrations(db)
		},
	}

	rollbackCmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last database migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, _ := database.InitDB()
			return RollbackLastMigration(db)
		},
	}

	migCmd.AddCommand(sqlCmd, upCmd, rollbackCmd)
	rootCmd.AddCommand(migCmd)

}
