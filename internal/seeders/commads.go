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
	var fromTable string
	var count int

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

  fake:datetime
  fake:date

Example:
  forge seed make users
  forge seed make users --type fixture
  forge seed make roles --type sql
  forge seed make --from-table users          # generate fixture from an existing table
  forge seed make --from-table users --count 50
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				name = strings.TrimSpace(args[0])
			} else if fromTable != "" {
				name = fromTable
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

			// Schema-derived fixture from an existing table.
			if fromTable != "" {
				db, err := database.InitDB()
				if err != nil {
					return fmt.Errorf("failed to initialize database: %v", err)
				}
				content, err := BuildFixtureFromTable(db, fromTable, count)
				if err != nil {
					return err
				}
				path, err := writeSeed(name, content)
				if err != nil {
					return err
				}
				fmt.Printf("Created %s (from table %q)\n", path, fromTable)
				return nil
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
	makeCmd.Flags().StringVar(&fromTable, "from-table", "", "Generate a fixture from an existing table's schema")
	makeCmd.Flags().IntVar(&count, "count", 10, "Number of rows to generate (with --from-table)")

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
