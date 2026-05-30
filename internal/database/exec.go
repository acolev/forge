package database

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gorm.io/gorm"
)

// ExecSQL runs a raw SQL statement, printing SELECT results as a table.
func ExecSQL(gdb *gorm.DB, query string) error {
	return ExecSQLFormat(gdb, query, "table")
}

// ExecSQLFormat runs a raw SQL statement. Read-style statements (SELECT, WITH, ...)
// are printed in the requested format (table|json|csv); everything else reports
// the number of affected rows.
func ExecSQLFormat(gdb *gorm.DB, query, format string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("empty SQL query")
	}
	if format == "" {
		format = "table"
	}

	if isReadQuery(query) {
		rows, err := gdb.Raw(query).Rows()
		if err != nil {
			return fmt.Errorf("query failed: %v", err)
		}
		defer rows.Close()
		return printRows(rows, format)
	}

	res := gdb.Exec(query)
	if res.Error != nil {
		return fmt.Errorf("exec failed: %v", res.Error)
	}
	fmt.Printf("OK, %d row(s) affected\n", res.RowsAffected)
	return nil
}

// isReadQuery reports whether the statement returns a result set.
func isReadQuery(query string) bool {
	upper := strings.ToUpper(strings.TrimSpace(query))
	for _, prefix := range []string{"SELECT", "WITH", "PRAGMA", "EXPLAIN", "SHOW"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

func printRows(rows *sql.Rows, format string) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("unable to read columns: %v", err)
	}

	var data [][]string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("unable to scan row: %v", err)
		}
		cells := make([]string, len(cols))
		for i, v := range vals {
			cells[i] = formatValue(v)
		}
		data = append(data, cells)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	switch strings.ToLower(format) {
	case "json":
		return printJSON(cols, data)
	case "csv":
		return printCSV(cols, data)
	case "table", "":
		return printTable(cols, data)
	default:
		return fmt.Errorf("unknown --format %q (use: table, json, csv)", format)
	}
}

func printTable(cols []string, data [][]string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(cols, "\t"))
	seps := make([]string, len(cols))
	for i, c := range cols {
		seps[i] = strings.Repeat("-", len(c))
	}
	fmt.Fprintln(w, strings.Join(seps, "\t"))
	for _, row := range data {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Printf("\n%d row(s)\n", len(data))
	return nil
}

func printJSON(cols []string, data [][]string) error {
	out := make([]map[string]string, 0, len(data))
	for _, row := range data {
		obj := make(map[string]string, len(cols))
		for i, c := range cols {
			obj[c] = row[i]
		}
		out = append(out, obj)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printCSV(cols []string, data [][]string) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.Write(cols); err != nil {
		return err
	}
	for _, row := range data {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func formatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}
