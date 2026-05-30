// Package schema provides driver-aware database introspection and rendering
// (SQL dump, human-readable tables, and ERD diagrams) for sqlite, postgres and
// mysql, plus a cross-driver "drop all tables" used by `forge db fresh`.
package schema

import (
	"fmt"
	"sort"

	"gorm.io/gorm"
)

// Model is a driver-neutral description of a database schema.
type Model struct {
	Driver string
	Tables []Table
}

type Table struct {
	Name        string
	Columns     []Column
	PrimaryKey  []string
	ForeignKeys []ForeignKey
	Indexes     []Index
}

type Column struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

type ForeignKey struct {
	Columns    []string
	RefTable   string
	RefColumns []string
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Table returns the table with the given name, or nil.
func (m *Model) Table(name string) *Table {
	for i := range m.Tables {
		if m.Tables[i].Name == name {
			return &m.Tables[i]
		}
	}
	return nil
}

// Introspect builds a Model from the live database connection.
func Introspect(db *gorm.DB) (*Model, error) {
	driver := db.Dialector.Name()
	switch driver {
	case "sqlite":
		return introspectSQLite(db)
	case "postgres":
		return introspectPostgres(db)
	case "mysql":
		return introspectMySQL(db)
	default:
		return nil, fmt.Errorf("schema introspection not supported for driver %q", driver)
	}
}

// DropAllTables removes every user table in the current database. Used by
// `forge db fresh`. Foreign-key enforcement is disabled for the duration so
// drop order does not matter.
func DropAllTables(db *gorm.DB) error {
	m, err := Introspect(db)
	if err != nil {
		return err
	}
	if len(m.Tables) == 0 {
		return nil
	}

	q := identQuoter(m.Driver)
	switch m.Driver {
	case "sqlite":
		if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
			return err
		}
		defer db.Exec("PRAGMA foreign_keys = ON")
		for _, t := range m.Tables {
			if err := db.Exec("DROP TABLE IF EXISTS " + q(t.Name)).Error; err != nil {
				return fmt.Errorf("drop %s: %w", t.Name, err)
			}
		}
	case "mysql":
		if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
			return err
		}
		defer db.Exec("SET FOREIGN_KEY_CHECKS = 1")
		for _, t := range m.Tables {
			if err := db.Exec("DROP TABLE IF EXISTS " + q(t.Name)).Error; err != nil {
				return fmt.Errorf("drop %s: %w", t.Name, err)
			}
		}
	case "postgres":
		for _, t := range m.Tables {
			if err := db.Exec("DROP TABLE IF EXISTS " + q(t.Name) + " CASCADE").Error; err != nil {
				return fmt.Errorf("drop %s: %w", t.Name, err)
			}
		}
	default:
		return fmt.Errorf("drop all not supported for driver %q", m.Driver)
	}
	return nil
}

// identQuoter returns a dialect-appropriate identifier quoter.
func identQuoter(driver string) func(string) string {
	if driver == "mysql" {
		return func(s string) string { return "`" + s + "`" }
	}
	return func(s string) string { return `"` + s + `"` }
}

func sortTables(tables []Table) {
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
}
