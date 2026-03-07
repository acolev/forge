package seeders

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
	var only string
	var seedType string

	seedCmd := &cobra.Command{Use: "seed", Short: "Seeders"}

	makeCmd := &cobra.Command{
		Use:   "make [name]",
		Short: "Create a YAML seed file",
		Long: `Create a YAML seed file scaffold.

Supported seed types:
  - fixture
  - sql
  - go

Fixture scaffolds support count/template generation and fake tokens.

Fake tokens:
  - fake:first_name
  - fake:last_name
  - fake:full_name
  - fake:email
  - fake:company
  - fake:phone
  - fake:sentence
  - fake:uuid
  - fake:bool
  - fake:int:min:max

Example:
  forge seed make users
  forge seed make users --type fixture
  forge seed make roles --type sql
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				name = strings.TrimSpace(args[0])
			} else {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter seed name: ")
				value, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read seed name: %w", err)
				}
				name = strings.TrimSpace(value)
				if name == "" {
					return errors.New("seed name cannot be empty")
				}
			}

			path, err := CreateSeed(name, seedType)
			if err != nil {
				return err
			}
			fmt.Printf("Created %s\n", path)
			return nil
		},
	}
	makeCmd.Flags().StringVar(&seedType, "type", "fixture", "Seed type: fixture, sql, go")

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
	runCmd.Flags().StringVar(&only, "only", "", "Comma-separated seeder names to run")

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

	seedCmd.AddCommand(makeCmd, upCmd, runCmd, statusCmd, resetCmd)
	rootCmd.AddCommand(seedCmd)
}
