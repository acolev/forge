package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// RenderText renders a human-readable overview of the schema.
func RenderText(m *Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Schema (%s) — %d table(s)\n", m.Driver, len(m.Tables))
	for _, t := range m.Tables {
		fmt.Fprintf(&b, "\n%s\n", t.Name)
		for _, c := range t.Columns {
			flags := []string{}
			if contains(t.PrimaryKey, c.Name) {
				flags = append(flags, "PK")
			}
			if fkCol(t, c.Name) {
				flags = append(flags, "FK")
			}
			if !c.Nullable {
				flags = append(flags, "NOT NULL")
			}
			suffix := ""
			if len(flags) > 0 {
				suffix = "  [" + strings.Join(flags, ", ") + "]"
			}
			fmt.Fprintf(&b, "  - %-24s %s%s\n", c.Name, c.Type, suffix)
		}
		for _, fk := range t.ForeignKeys {
			fmt.Fprintf(&b, "  FK (%s) -> %s(%s)\n",
				strings.Join(fk.Columns, ", "), fk.RefTable, strings.Join(fk.RefColumns, ", "))
		}
		for _, ix := range t.Indexes {
			kind := "INDEX"
			if ix.Unique {
				kind = "UNIQUE"
			}
			fmt.Fprintf(&b, "  %s %s (%s)\n", kind, ix.Name, strings.Join(ix.Columns, ", "))
		}
	}
	return b.String()
}

// RenderMermaid renders the schema as a Mermaid erDiagram (renders natively on
// GitHub and most Markdown viewers, no external tooling required).
func RenderMermaid(m *Model) string {
	var b strings.Builder
	b.WriteString("erDiagram\n")
	for _, t := range m.Tables {
		fmt.Fprintf(&b, "    %s {\n", mermaidIdent(t.Name))
		for _, c := range t.Columns {
			key := ""
			switch {
			case contains(t.PrimaryKey, c.Name):
				key = "PK"
			case fkCol(t, c.Name):
				key = "FK"
			}
			line := fmt.Sprintf("        %s %s", mermaidType(c.Type), c.Name)
			if key != "" {
				line += " " + key
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("    }\n")
	}
	for _, t := range m.Tables {
		for _, fk := range t.ForeignKeys {
			label := strings.Join(fk.Columns, ",")
			// child }o--|| parent : "fk columns"
			fmt.Fprintf(&b, "    %s }o--|| %s : \"%s\"\n",
				mermaidIdent(t.Name), mermaidIdent(fk.RefTable), label)
		}
	}
	return b.String()
}

// RenderDOT renders the schema as a Graphviz DOT graph.
func RenderDOT(m *Model) string {
	var b strings.Builder
	b.WriteString("digraph schema {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=record, fontname=\"Helvetica\"];\n")
	for _, t := range m.Tables {
		var fields []string
		for _, c := range t.Columns {
			tag := c.Name
			if contains(t.PrimaryKey, c.Name) {
				tag = c.Name + " (PK)"
			} else if fkCol(t, c.Name) {
				tag = c.Name + " (FK)"
			}
			fields = append(fields, dotEscape(tag+" : "+c.Type))
		}
		fmt.Fprintf(&b, "  %q [label=\"{%s|%s}\"];\n", t.Name, t.Name, strings.Join(fields, "|"))
	}
	for _, t := range m.Tables {
		for _, fk := range t.ForeignKeys {
			fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", t.Name, fk.RefTable, strings.Join(fk.Columns, ","))
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// DumpSQL returns DDL for the schema. sqlite and mysql expose authoritative
// native DDL; postgres is reconstructed from the introspected model.
func DumpSQL(db *gorm.DB, m *Model) (string, error) {
	switch m.Driver {
	case "sqlite":
		allow := map[string]bool{}
		for _, t := range m.Tables {
			allow[t.Name] = true
		}
		type ddlRow struct {
			SQL     string `gorm:"column:sql"`
			TblName string `gorm:"column:tbl_name"`
		}
		var rows []ddlRow
		if err := db.Raw(
			`SELECT sql, tbl_name FROM sqlite_master
			 WHERE sql IS NOT NULL AND type IN ('table','index') AND name NOT LIKE 'sqlite_%'
			 ORDER BY type DESC, name`,
		).Scan(&rows).Error; err != nil {
			return "", err
		}
		var stmts []string
		for _, r := range rows {
			if allow[r.TblName] {
				stmts = append(stmts, r.SQL)
			}
		}
		if len(stmts) == 0 {
			return "", nil
		}
		return strings.Join(stmts, ";\n\n") + ";\n", nil

	case "mysql":
		var b strings.Builder
		for _, t := range m.Tables {
			type row struct {
				Table  string `gorm:"column:Table"`
				Create string `gorm:"column:Create Table"`
			}
			var r row
			if err := db.Raw("SHOW CREATE TABLE " + identQuoter("mysql")(t.Name)).Scan(&r).Error; err != nil {
				return "", err
			}
			b.WriteString(r.Create)
			b.WriteString(";\n\n")
		}
		return b.String(), nil

	case "postgres":
		return reconstructPostgresDDL(m), nil

	default:
		return "", fmt.Errorf("SQL dump not supported for driver %q", m.Driver)
	}
}

func reconstructPostgresDDL(m *Model) string {
	q := identQuoter("postgres")
	var b strings.Builder
	for _, t := range m.Tables {
		fmt.Fprintf(&b, "CREATE TABLE %s (\n", q(t.Name))
		var lines []string
		for _, c := range t.Columns {
			line := "    " + q(c.Name) + " " + c.Type
			if !c.Nullable {
				line += " NOT NULL"
			}
			if c.Default != "" {
				line += " DEFAULT " + c.Default
			}
			lines = append(lines, line)
		}
		if len(t.PrimaryKey) > 0 {
			cols := make([]string, len(t.PrimaryKey))
			for i, c := range t.PrimaryKey {
				cols[i] = q(c)
			}
			lines = append(lines, "    PRIMARY KEY ("+strings.Join(cols, ", ")+")")
		}
		for _, fk := range t.ForeignKeys {
			from := make([]string, len(fk.Columns))
			for i, c := range fk.Columns {
				from[i] = q(c)
			}
			to := make([]string, len(fk.RefColumns))
			for i, c := range fk.RefColumns {
				to[i] = q(c)
			}
			lines = append(lines, fmt.Sprintf("    FOREIGN KEY (%s) REFERENCES %s (%s)",
				strings.Join(from, ", "), q(fk.RefTable), strings.Join(to, ", ")))
		}
		b.WriteString(strings.Join(lines, ",\n"))
		b.WriteString("\n);\n")
		for _, ix := range t.Indexes {
			kind := "INDEX"
			if ix.Unique {
				kind = "UNIQUE INDEX"
			}
			cols := make([]string, len(ix.Columns))
			for i, c := range ix.Columns {
				cols[i] = q(c)
			}
			fmt.Fprintf(&b, "CREATE %s %s ON %s (%s);\n", kind, q(ix.Name), q(t.Name), strings.Join(cols, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ---------------- helpers ----------------

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func fkCol(t Table, col string) bool {
	for _, fk := range t.ForeignKeys {
		if contains(fk.Columns, col) {
			return true
		}
	}
	return false
}

// mermaidIdent strips characters Mermaid entity names cannot contain.
func mermaidIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// mermaidType collapses whitespace so a type stays a single token.
func mermaidType(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	return strings.ReplaceAll(s, " ", "_")
}

func dotEscape(s string) string {
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "<", "\\<")
	s = strings.ReplaceAll(s, ">", "\\>")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
