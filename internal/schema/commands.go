package schema

import (
	"fmt"
	"os"

	"forge/internal/database"

	"github.com/spf13/cobra"
)

// internalTables are Forge's own bookkeeping tables, hidden from schema output
// unless --all is passed.
var internalTables = map[string]bool{"migrations": true, "seeds": true}

// Register attaches schema:* subcommands to the given parent command (the `db` group).
func Register(parent *cobra.Command) {
	parent.AddCommand(showCmd())
	parent.AddCommand(dumpCmd())
	parent.AddCommand(erdCmd())
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
