package seeders

import (
	"fmt"
	"forge/internal/database"
	"github.com/spf13/cobra"
	"strings"
)

func RegisterCommands(rootCmd *cobra.Command) {
	var only string

	seedCmd := &cobra.Command{Use: "seed", Short: "Seeders"}

	upCmd := &cobra.Command{
		Use: "up", Short: "Run all pending seeders",
		RunE: func(*cobra.Command, []string) error {
			db, err := database.InitDB()
			if err != nil {
				return err
			}
			return ApplyAll(db)
		},
	}

	runCmd := &cobra.Command{
		Use: "run", Short: "Run specific seeders",
		RunE: func(*cobra.Command, []string) error {
			if only == "" {
				return fmt.Errorf("--only=plans,tenants_accounts_users")
			}
			db, err := database.InitDB()
			if err != nil {
				return err
			}
			return ApplyOnly(db, strings.Split(only, ","))
		},
	}

	statusCmd := &cobra.Command{
		Use: "status", Short: "Show executed seeders",
		RunE: func(*cobra.Command, []string) error {
			db, err := database.InitDB()
			if err != nil {
				return err
			}
			return Status(db)
		},
	}

	resetCmd := &cobra.Command{
		Use: "reset", Short: "Clear seed status (does not delete data)",
		RunE: func(*cobra.Command, []string) error {
			db, err := database.InitDB()
			if err != nil {
				return err
			}
			return Reset(db)
		},
	}

	seedCmd.AddCommand(upCmd, runCmd, statusCmd, resetCmd)
	rootCmd.AddCommand(seedCmd)
}
