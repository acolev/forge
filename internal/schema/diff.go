package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// SnapshotJSON serializes a model to stable, indented JSON.
func SnapshotJSON(m *Model) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// LoadSnapshot reads a model previously written by SnapshotJSON.
func LoadSnapshot(path string) (*Model, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Model
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("invalid snapshot %s: %w", path, err)
	}
	return &m, nil
}

// ColumnChange describes a single altered column attribute.
type ColumnChange struct {
	Column string
	Field  string // "type", "nullable", "default"
	Old    string
	New    string
}

// TableDiff captures changes within one table that exists on both sides.
type TableDiff struct {
	Name           string
	AddedColumns   []Column
	RemovedColumns []string
	ChangedColumns []ColumnChange
	Notes          []string // PK / index / FK changes
}

func (t TableDiff) empty() bool {
	return len(t.AddedColumns) == 0 && len(t.RemovedColumns) == 0 &&
		len(t.ChangedColumns) == 0 && len(t.Notes) == 0
}

// Diff is the full comparison between two models (old -> new).
type Diff struct {
	AddedTables   []string
	RemovedTables []string
	ChangedTables []TableDiff
}

// Empty reports whether the two models are equivalent.
func (d Diff) Empty() bool {
	return len(d.AddedTables) == 0 && len(d.RemovedTables) == 0 && len(d.ChangedTables) == 0
}

// DiffModels compares oldM (e.g. a snapshot) against newM (e.g. the live DB).
func DiffModels(oldM, newM *Model) Diff {
	var d Diff
	oldT := tableMap(oldM)
	newT := tableMap(newM)

	for name := range newT {
		if _, ok := oldT[name]; !ok {
			d.AddedTables = append(d.AddedTables, name)
		}
	}
	for name := range oldT {
		if _, ok := newT[name]; !ok {
			d.RemovedTables = append(d.RemovedTables, name)
		}
	}
	sortStrings(d.AddedTables)
	sortStrings(d.RemovedTables)

	for _, nt := range newM.Tables {
		ot, ok := oldT[nt.Name]
		if !ok {
			continue
		}
		if td := diffTable(ot, nt); !td.empty() {
			d.ChangedTables = append(d.ChangedTables, td)
		}
	}
	sort.Slice(d.ChangedTables, func(i, j int) bool { return d.ChangedTables[i].Name < d.ChangedTables[j].Name })
	return d
}

func diffTable(oldT, newT Table) TableDiff {
	td := TableDiff{Name: newT.Name}
	oldC := columnMap(oldT)
	newC := columnMap(newT)

	for _, c := range newT.Columns {
		if _, ok := oldC[c.Name]; !ok {
			td.AddedColumns = append(td.AddedColumns, c)
		}
	}
	for _, c := range oldT.Columns {
		if _, ok := newC[c.Name]; !ok {
			td.RemovedColumns = append(td.RemovedColumns, c.Name)
		}
	}
	for _, nc := range newT.Columns {
		oc, ok := oldC[nc.Name]
		if !ok {
			continue
		}
		if !strings.EqualFold(oc.Type, nc.Type) {
			td.ChangedColumns = append(td.ChangedColumns, ColumnChange{nc.Name, "type", oc.Type, nc.Type})
		}
		if oc.Nullable != nc.Nullable {
			td.ChangedColumns = append(td.ChangedColumns, ColumnChange{nc.Name, "nullable", boolStr(oc.Nullable), boolStr(nc.Nullable)})
		}
		if oc.Default != nc.Default {
			td.ChangedColumns = append(td.ChangedColumns, ColumnChange{nc.Name, "default", oc.Default, nc.Default})
		}
	}

	if !equalStrings(oldT.PrimaryKey, newT.PrimaryKey) {
		td.Notes = append(td.Notes, fmt.Sprintf("primary key: (%s) -> (%s)",
			strings.Join(oldT.PrimaryKey, ", "), strings.Join(newT.PrimaryKey, ", ")))
	}
	td.Notes = append(td.Notes, setDiff("index", indexSigs(oldT), indexSigs(newT))...)
	td.Notes = append(td.Notes, setDiff("foreign key", fkSigs(oldT), fkSigs(newT))...)
	return td
}

// RenderDiff produces a human-readable, +/-/~ formatted diff.
func RenderDiff(d Diff) string {
	if d.Empty() {
		return "No differences — live schema matches the snapshot.\n"
	}
	var b strings.Builder
	for _, t := range d.AddedTables {
		fmt.Fprintf(&b, "+ table %s\n", t)
	}
	for _, t := range d.RemovedTables {
		fmt.Fprintf(&b, "- table %s\n", t)
	}
	for _, td := range d.ChangedTables {
		fmt.Fprintf(&b, "~ table %s\n", td.Name)
		for _, c := range td.AddedColumns {
			fmt.Fprintf(&b, "    + column %s %s\n", c.Name, c.Type)
		}
		for _, c := range td.RemovedColumns {
			fmt.Fprintf(&b, "    - column %s\n", c)
		}
		for _, c := range td.ChangedColumns {
			fmt.Fprintf(&b, "    ~ column %s: %s %q -> %q\n", c.Column, c.Field, c.Old, c.New)
		}
		for _, n := range td.Notes {
			fmt.Fprintf(&b, "    ~ %s\n", n)
		}
	}
	return b.String()
}

// ---------------- helpers ----------------

func tableMap(m *Model) map[string]Table {
	out := map[string]Table{}
	if m == nil {
		return out
	}
	for _, t := range m.Tables {
		out[t.Name] = t
	}
	return out
}

func columnMap(t Table) map[string]Column {
	out := map[string]Column{}
	for _, c := range t.Columns {
		out[c.Name] = c
	}
	return out
}

func indexSigs(t Table) map[string]bool {
	out := map[string]bool{}
	for _, ix := range t.Indexes {
		kind := "index"
		if ix.Unique {
			kind = "unique"
		}
		out[fmt.Sprintf("%s (%s)", kind, strings.Join(ix.Columns, ","))] = true
	}
	return out
}

func fkSigs(t Table) map[string]bool {
	out := map[string]bool{}
	for _, fk := range t.ForeignKeys {
		out[fmt.Sprintf("(%s) -> %s(%s)", strings.Join(fk.Columns, ","), fk.RefTable, strings.Join(fk.RefColumns, ","))] = true
	}
	return out
}

func setDiff(label string, oldS, newS map[string]bool) []string {
	var notes []string
	var added, removed []string
	for s := range newS {
		if !oldS[s] {
			added = append(added, s)
		}
	}
	for s := range oldS {
		if !newS[s] {
			removed = append(removed, s)
		}
	}
	sortStrings(added)
	sortStrings(removed)
	for _, s := range added {
		notes = append(notes, fmt.Sprintf("+ %s %s", label, s))
	}
	for _, s := range removed {
		notes = append(notes, fmt.Sprintf("- %s %s", label, s))
	}
	return notes
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
