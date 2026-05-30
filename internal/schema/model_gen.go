package schema

import (
	"fmt"
	"sort"
	"strings"
)

// RenderGoModels renders Go structs (with gorm/json tags) for the given tables.
// If tables is empty, every table in the model is rendered.
func RenderGoModels(m *Model, pkg string, tables []string) string {
	if pkg == "" {
		pkg = "models"
	}
	want := map[string]bool{}
	for _, t := range tables {
		want[t] = true
	}

	var bodies []string
	needsTime := false
	for _, t := range m.Tables {
		if len(want) > 0 && !want[t.Name] {
			continue
		}
		body, usesTime := renderStruct(t)
		if usesTime {
			needsTime = true
		}
		bodies = append(bodies, body)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	if needsTime {
		b.WriteString("import \"time\"\n\n")
	}
	b.WriteString(strings.Join(bodies, "\n"))
	return b.String()
}

func renderStruct(t Table) (string, bool) {
	usesTime := false
	pk := map[string]bool{}
	for _, c := range t.PrimaryKey {
		pk[c] = true
	}

	// Align field/type columns for readability.
	type field struct{ name, typ, tag string }
	var fields []field
	nameW, typeW := 0, 0
	for _, c := range t.Columns {
		goT := goType(c, pk[c.Name])
		if goT == "time.Time" || goT == "*time.Time" {
			usesTime = true
		}
		fn := goFieldName(c.Name)
		tag := buildTag(c, pk[c.Name])
		fields = append(fields, field{fn, goT, tag})
		if len(fn) > nameW {
			nameW = len(fn)
		}
		if len(goT) > typeW {
			typeW = len(goT)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "type %s struct {\n", goStructName(t.Name))
	for _, f := range fields {
		fmt.Fprintf(&b, "\t%-*s %-*s %s\n", nameW, f.name, typeW, f.typ, f.tag)
	}
	b.WriteString("}\n")
	fmt.Fprintf(&b, "\nfunc (%s) TableName() string { return %q }\n", goStructName(t.Name), t.Name)
	return b.String(), usesTime
}

func buildTag(c Column, isPK bool) string {
	gormParts := []string{"column:" + c.Name}
	if isPK {
		gormParts = append(gormParts, "primaryKey")
	}
	if !c.Nullable && !isPK {
		gormParts = append(gormParts, "not null")
	}
	return fmt.Sprintf("`gorm:%q json:%q`", strings.Join(gormParts, ";"), c.Name)
}

// goType maps a SQL column type to a Go type. Nullable non-PK columns become
// pointers so NULL is representable.
func goType(c Column, isPK bool) string {
	base := goBaseType(c.Type)
	if isPK && (base == "int64") {
		base = "uint"
	}
	if c.Nullable && !isPK && base != "[]byte" {
		return "*" + base
	}
	return base
}

func goBaseType(sqlType string) string {
	t := strings.ToLower(strings.TrimSpace(sqlType))
	switch {
	case strings.Contains(t, "bool"), t == "tinyint(1)":
		return "bool"
	case strings.Contains(t, "serial"), strings.Contains(t, "int"):
		return "int64"
	case strings.Contains(t, "float"), strings.Contains(t, "double"),
		strings.Contains(t, "real"), strings.Contains(t, "numeric"), strings.Contains(t, "decimal"):
		return "float64"
	case strings.Contains(t, "date"), strings.Contains(t, "time"):
		return "time.Time"
	case strings.Contains(t, "blob"), strings.Contains(t, "bytea"), strings.Contains(t, "binary"):
		return "[]byte"
	default:
		// text, varchar, char, json, uuid, enum, ...
		return "string"
	}
}

// commonInitialisms get fully upper-cased in Go field names.
var commonInitialisms = map[string]bool{
	"id": true, "url": true, "uri": true, "api": true, "uuid": true,
	"ip": true, "http": true, "https": true, "json": true, "sql": true, "db": true,
}

func goFieldName(col string) string {
	parts := strings.FieldsFunc(col, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if commonInitialisms[strings.ToLower(p)] {
			b.WriteString(strings.ToUpper(p))
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

func goStructName(table string) string {
	return goFieldName(singularize(table))
}

// singularize is a small, deliberately naive English singularizer.
func singularize(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "ies"):
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(lower, "ses"), strings.HasSuffix(lower, "xes"),
		strings.HasSuffix(lower, "zes"), strings.HasSuffix(lower, "ches"), strings.HasSuffix(lower, "shes"):
		return s[:len(s)-2]
	case strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss"):
		return s[:len(s)-1]
	default:
		return s
	}
}

func sortStrings(ss []string) { sort.Strings(ss) }
