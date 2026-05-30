package migrations

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"forge/internal/database"
	"forge/internal/schema"

	"github.com/spf13/cobra"
)

func RegisterCommands(rootCmd *cobra.Command) {

	migCmd := &cobra.Command{Use: "db", Short: "Database migration & schema commands"}

	migCmd.AddCommand(
		makeSQLCmd(),
		migrateCmd(),
		rollbackCmd(),
		resetCmd(),
		refreshCmd(),
		freshCmd(),
		statusCmd(),
		execCmd(),
	)

	// schema:dump / schema:show / schema:erd
	schema.Register(migCmd)

	rootCmd.AddCommand(migCmd)
}

func makeSQLCmd() *cobra.Command {
	return &cobra.Command{
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
  forge db make:sql create_table_users
  forge db make:sql add_column_users_email
  forge db make:sql
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
}

func migrateCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "migrate",
		Short: "Run pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			if dryRun {
				names, sqls, err := PendingUpSQL(db)
				if err != nil {
					return err
				}
				if len(names) == 0 {
					fmt.Println("No pending migrations.")
					return nil
				}
				fmt.Printf("-- %d pending migration(s) (dry run, nothing applied)\n", len(names))
				for i, n := range names {
					fmt.Printf("\n-- %s\n%s\n", n, sqls[i])
				}
				return nil
			}
			return RunMigrations(db)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print the SQL that would run without applying it")
	return c
}

func rollbackCmd() *cobra.Command {
	var step int
	c := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last database migration batch (use --step N for more)",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			return RollbackBatches(db, step)
		},
	}
	c.Flags().IntVar(&step, "step", 1, "number of batches to roll back")
	return c
}

func resetCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "reset",
		Short: "Roll back ALL migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmDestructive("This will roll back ALL migrations.", force) {
				return nil
			}
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			return ResetMigrations(db)
		},
	}
	c.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return c
}

func refreshCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "refresh",
		Short: "Roll back ALL migrations and run them again",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmDestructive("This will roll back ALL migrations and re-apply them.", force) {
				return nil
			}
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			if err := ResetMigrations(db); err != nil {
				return err
			}
			return RunMigrations(db)
		},
	}
	c.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return c
}

func freshCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "fresh",
		Short: "Drop ALL tables and run every migration from scratch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmDestructive("This will DROP ALL TABLES and re-run every migration.", force) {
				return nil
			}
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			if err := schema.DropAllTables(db); err != nil {
				return fmt.Errorf("drop all tables failed: %w", err)
			}
			fmt.Println("Dropped all tables.")
			return RunMigrations(db)
		},
	}
	c.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return c
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show applied and pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			rows, err := GetStatus(db)
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				fmt.Println("No migration files found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "STATUS\tBATCH\tMIGRATION")
			fmt.Fprintln(w, "------\t-----\t---------")
			pending := 0
			for _, r := range rows {
				if r.Applied {
					fmt.Fprintf(w, "applied\t%d\t%s\n", r.Batch, r.FileName)
				} else {
					pending++
					fmt.Fprintf(w, "pending\t-\t%s\n", r.FileName)
				}
			}
			w.Flush()
			fmt.Printf("\n%d applied, %d pending\n", len(rows)-pending, pending)
			return nil
		},
	}
}

func execCmd() *cobra.Command {
	var file, format string
	c := &cobra.Command{
		Use:   "exec [sql]",
		Short: "Execute a raw SQL query against the configured database",
		Example: `  forge db exec "SELECT * FROM users"
  forge db exec --file migration.sql
  echo "SELECT 1" | forge db exec -
  forge db exec --format json "SELECT * FROM users"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := readQuery(args, file)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("no SQL provided (pass a query, --file, or '-' for stdin)")
			}

			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			return database.ExecSQLFormat(db, query, format)
		},
	}
	c.Flags().StringVar(&file, "file", "", "read SQL from a file")
	c.Flags().StringVar(&format, "format", "table", "result format for SELECT: table | json | csv")
	return c
}

// readQuery resolves the SQL source: --file, stdin ("-"), or positional args.
func readQuery(args []string, file string) (string, error) {
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("unable to read %s: %w", file, err)
		}
		return string(b), nil
	}
	if len(args) == 1 && args[0] == "-" {
		b, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return "", fmt.Errorf("unable to read stdin: %w", err)
		}
		return string(b), nil
	}
	if len(args) == 0 {
		return "", errors.New("you must provide a SQL query, --file, or '-'")
	}
	return strings.Join(args, " "), nil
}

// confirmDestructive prompts for a yes/no confirmation unless force is set.
func confirmDestructive(message string, force bool) bool {
	if force {
		return true
	}
	fmt.Printf("%s\nContinue? [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "y" || answer == "yes" {
		return true
	}
	fmt.Println("Aborted.")
	return false
}
