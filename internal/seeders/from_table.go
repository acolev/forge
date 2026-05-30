package seeders

import (
	"fmt"
	"strings"

	"forge/internal/schema"

	"gorm.io/gorm"
)

// BuildFixtureFromTable introspects an existing table and renders a fixture
// seed YAML whose template matches the real columns, mapping each to a sensible
// fake token (and foreign keys to a $ref on the parent table).
func BuildFixtureFromTable(db *gorm.DB, table string, count int) (string, error) {
	m, err := schema.Introspect(db)
	if err != nil {
		return "", err
	}
	t := m.Table(table)
	if t == nil {
		return "", fmt.Errorf("table %q not found in current database", table)
	}
	if count <= 0 {
		count = 10
	}

	// Map each FK column to its parent (table, column).
	fkByCol := map[string]struct{ table, col string }{}
	for _, fk := range t.ForeignKeys {
		for i, c := range fk.Columns {
			ref := struct{ table, col string }{fk.RefTable, "id"}
			if i < len(fk.RefColumns) {
				ref.col = fk.RefColumns[i]
			}
			fkByCol[c] = ref
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", table)
	b.WriteString("type: fixture\n")
	fmt.Fprintf(&b, "table: %s\n", table)
	fmt.Fprintf(&b, "count: %d\n", count)
	fmt.Fprintf(&b, "# Generated from table %q — adjust fake tokens / $ref filters as needed.\n", table)
	b.WriteString("template:\n")

	wrote := 0
	for _, c := range t.Columns {
		if skipColumn(t, c) {
			continue
		}
		value, comment := columnSeedValue(c, fkByCol)
		line := fmt.Sprintf("  %s: %q", c.Name, value)
		if comment != "" {
			line += "   # " + comment
		}
		b.WriteString(line + "\n")
		wrote++
	}
	if wrote == 0 {
		b.WriteString("  # (no fillable columns detected; add fields manually)\n")
	}
	return b.String(), nil
}

// skipColumn drops auto primary keys and framework timestamp columns — the DB
// assigns those.
func skipColumn(t *schema.Table, c schema.Column) bool {
	name := strings.ToLower(c.Name)
	switch name {
	case "created_at", "updated_at", "deleted_at":
		return true
	}
	// Single integer primary key → auto-increment, let the DB assign it.
	if len(t.PrimaryKey) == 1 && t.PrimaryKey[0] == c.Name && isIntType(c.Type) {
		return true
	}
	return false
}

// columnSeedValue returns the YAML value (a fake token or $ref shortcut) and an
// optional trailing comment for a column.
func columnSeedValue(c schema.Column, fkByCol map[string]struct{ table, col string }) (string, string) {
	if ref, ok := fkByCol[c.Name]; ok {
		// ref:<table>|<col>=1|<col> — resolves to the first parent row; tweak the filter.
		return fmt.Sprintf("ref:%s|%s=1|%s", ref.table, ref.col, ref.col),
			fmt.Sprintf("FK -> %s.%s", ref.table, ref.col)
	}

	name := strings.ToLower(c.Name)
	typ := strings.ToLower(c.Type)

	switch {
	case name == "email" || strings.HasSuffix(name, "_email"):
		return "fake:email", ""
	case name == "first_name":
		return "fake:first_name", ""
	case name == "last_name":
		return "fake:last_name", ""
	case name == "name" || name == "full_name" || strings.HasSuffix(name, "_name"):
		return "fake:full_name", ""
	case strings.Contains(name, "phone"):
		return "fake:phone", ""
	case strings.Contains(name, "company"):
		return "fake:company", ""
	case strings.Contains(name, "uuid") || strings.Contains(typ, "uuid"):
		return "fake:uuid", ""
	case name == "title" || name == "body" || strings.Contains(name, "description") ||
		strings.Contains(name, "bio") || strings.Contains(name, "comment"):
		return "fake:sentence", ""
	}

	switch {
	case isBoolType(typ) || isBoolName(name):
		return "fake:bool", ""
	case isIntType(typ):
		return "fake:int:1:100", ""
	case isDateType(typ):
		return "fake:datetime", ""
	default:
		return "fake:sentence", fmt.Sprintf("type: %s", c.Type)
	}
}

func isIntType(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "int") || strings.Contains(t, "serial")
}

func isBoolType(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "bool") || t == "tinyint(1)"
}

func isBoolName(name string) bool {
	return strings.HasPrefix(name, "is_") || strings.HasPrefix(name, "has_") || strings.HasPrefix(name, "can_")
}

func isDateType(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "date") || strings.Contains(t, "time")
}
