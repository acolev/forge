package schema

import (
	"fmt"
	"os"
	"strings"

	"forge/internal/database"

	"github.com/spf13/cobra"
)

// internalTables are Forge's own bookkeeping tables, hidden from schema output
// unless --all is passed.
var internalTables = map[string]bool{"migrations": true, "seeds": true}

// defaultSnapshotPath is where schema:snapshot writes and schema:diff reads.
const defaultSnapshotPath = "database/schema.snapshot.json"

// Register attaches schema:* subcommands to the given parent command (the `db` group).
func Register(parent *cobra.Command) {
	parent.AddCommand(showCmd())
	parent.AddCommand(dumpCmd())
	parent.AddCommand(erdCmd())
	parent.AddCommand(snapshotCmd())
	parent.AddCommand(diffCmd())
	parent.AddCommand(modelCmd())
}

func showCmd() *cobra.Command {
	var all bool
	c := &cobra.Command{
		Use:   "schema:show",
		Short: "Print a human-readable overview of the current database schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := introspect(all)
			if err != nil {
				return err
			}
			fmt.Print(RenderText(m))
			return nil
		},
	}
	c.Flags().BoolVarP(&all, "all", "a", false, "include Forge's internal tables (migrations, seeds)")
	return c
}

func dumpCmd() *cobra.Command {
	var out string
	var all bool
	c := &cobra.Command{
		Use:   "schema:dump",
		Short: "Dump the current database schema as SQL DDL",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			m, err := Introspect(db)
			if err != nil {
				return err
			}
			m = applyVisibility(m, all)
			ddl, err := DumpSQL(db, m)
			if err != nil {
				return err
			}
			return writeOut(out, ddl)
		},
	}
	c.Flags().StringVarP(&out, "out", "o", "", "write to file instead of stdout")
	c.Flags().BoolVarP(&all, "all", "a", false, "include Forge's internal tables (migrations, seeds)")
	return c
}

func erdCmd() *cobra.Command {
	var out, format string
	var all bool
	c := &cobra.Command{
		Use:   "schema:erd",
		Short: "Generate an ERD from the current database schema (Mermaid or Graphviz DOT)",
		Example: `  forge db schema:erd                  # Mermaid to stdout
  forge db schema:erd -o erd.mmd       # Mermaid to file
  forge db schema:erd --format dot -o erd.dot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := introspect(all)
			if err != nil {
				return err
			}
			var rendered string
			switch format {
			case "mermaid", "":
				rendered = RenderMermaid(m)
			case "dot", "graphviz":
				rendered = RenderDOT(m)
			default:
				return fmt.Errorf("unknown --format %q (use: mermaid, dot)", format)
			}
			return writeOut(out, rendered)
		},
	}
	c.Flags().StringVarP(&out, "out", "o", "", "write to file instead of stdout")
	c.Flags().StringVarP(&format, "format", "f", "mermaid", "diagram format: mermaid | dot")
	c.Flags().BoolVarP(&all, "all", "a", false, "include Forge's internal tables (migrations, seeds)")
	return c
}

func snapshotCmd() *cobra.Command {
	var out string
	var all bool
	c := &cobra.Command{
		Use:   "schema:snapshot",
		Short: "Save the current database schema to a snapshot file (for schema:diff)",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := introspect(all)
			if err != nil {
				return err
			}
			data, err := SnapshotJSON(m)
			if err != nil {
				return err
			}
			if mkErr := os.MkdirAll(dirOf(out), 0o755); mkErr != nil {
				return mkErr
			}
			if err := os.WriteFile(out, data, 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", out, err)
			}
			fmt.Printf("Wrote snapshot %s (%d table(s))\n", out, len(m.Tables))
			return nil
		},
	}
	c.Flags().StringVarP(&out, "out", "o", defaultSnapshotPath, "snapshot file path")
	c.Flags().BoolVarP(&all, "all", "a", false, "include Forge's internal tables (migrations, seeds)")
	return c
}

func diffCmd() *cobra.Command {
	var from string
	var all, exitCode bool
	c := &cobra.Command{
		Use:   "schema:diff",
		Short: "Diff the live database schema against a saved snapshot",
		Long: `Compare the current database schema against a snapshot created with
schema:snapshot. Useful for detecting schema drift (e.g. in CI with --exit-code).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			oldM, err := LoadSnapshot(from)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("snapshot %s not found — run `forge db schema:snapshot` first", from)
				}
				return err
			}
			newM, err := introspect(all)
			if err != nil {
				return err
			}
			d := DiffModels(oldM, newM)
			fmt.Print(RenderDiff(d))
			if exitCode && !d.Empty() {
				os.Exit(1)
			}
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", defaultSnapshotPath, "snapshot file to compare against")
	c.Flags().BoolVarP(&all, "all", "a", false, "include Forge's internal tables (migrations, seeds)")
	c.Flags().BoolVar(&exitCode, "exit-code", false, "exit with code 1 if the schema differs (for CI)")
	return c
}

func modelCmd() *cobra.Command {
	var out, pkg string
	var all bool
	c := &cobra.Command{
		Use:   "make:model [table]",
		Short: "Generate Go structs from the database schema",
		Long: `Generate Go model structs (with gorm/json tags) from existing tables.

The generated .go file is plain source for YOUR project — Forge writes it, your
app compiles it. Pass a table name to generate a single model, or omit it for all.`,
		Example: `  forge db make:model users -o models/user.go
  forge db make:model --package entities -o models/models.go`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.InitDB()
			if err != nil {
				return fmt.Errorf("failed to initialize database: %v", err)
			}
			m, err := Introspect(db)
			if err != nil {
				return err
			}

			var tables []string
			if len(args) > 0 && args[0] != "" {
				// Explicit table — honor it even if it's an internal table.
				tables = []string{args[0]}
				if m.Table(args[0]) == nil {
					return fmt.Errorf("table %q not found in current database", args[0])
				}
			} else {
				m = applyVisibility(m, all)
			}

			code := RenderGoModels(m, pkg, tables)
			return writeOut(out, code)
		},
	}
	c.Flags().StringVarP(&out, "out", "o", "", "write to file instead of stdout")
	c.Flags().StringVarP(&pkg, "package", "p", "models", "Go package name for the generated file")
	c.Flags().BoolVarP(&all, "all", "a", false, "include Forge's internal tables when generating all models")
	return c
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i > 0 {
		return path[:i]
	}
	return "."
}

func introspect(all bool) (*Model, error) {
	db, err := database.InitDB()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}
	m, err := Introspect(db)
	if err != nil {
		return nil, err
	}
	return applyVisibility(m, all), nil
}

// applyVisibility drops Forge's internal tables unless all is true.
func applyVisibility(m *Model, all bool) *Model {
	if all {
		return m
	}
	filtered := &Model{Driver: m.Driver}
	for _, t := range m.Tables {
		if internalTables[t.Name] {
			continue
		}
		filtered.Tables = append(filtered.Tables, t)
	}
	return filtered
}

func writeOut(path, content string) error {
	if path == "" {
		fmt.Print(content)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	fmt.Printf("Wrote %s\n", path)
	return nil
}
