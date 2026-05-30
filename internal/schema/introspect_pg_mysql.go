package schema

import (
	"strings"

	"gorm.io/gorm"
)

// ---------------- Postgres ----------------

func introspectPostgres(db *gorm.DB) (*Model, error) {
	m := &Model{Driver: "postgres"}

	var tableNames []string
	if err := db.Raw(
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'
		 ORDER BY table_name`,
	).Scan(&tableNames).Error; err != nil {
		return nil, err
	}

	for _, name := range tableNames {
		t := Table{Name: name}

		type colRow struct {
			Name     string  `gorm:"column:column_name"`
			Type     string  `gorm:"column:data_type"`
			Nullable string  `gorm:"column:is_nullable"`
			Default  *string `gorm:"column:column_default"`
		}
		var cols []colRow
		if err := db.Raw(
			`SELECT column_name, data_type, is_nullable, column_default
			 FROM information_schema.columns
			 WHERE table_schema = current_schema() AND table_name = ?
			 ORDER BY ordinal_position`, name,
		).Scan(&cols).Error; err != nil {
			return nil, err
		}
		for _, c := range cols {
			def := ""
			if c.Default != nil {
				def = *c.Default
			}
			t.Columns = append(t.Columns, Column{
				Name:     c.Name,
				Type:     c.Type,
				Nullable: strings.EqualFold(c.Nullable, "YES"),
				Default:  def,
			})
		}

		var pk []string
		if err := db.Raw(
			`SELECT kcu.column_name
			 FROM information_schema.table_constraints tc
			 JOIN information_schema.key_column_usage kcu
			   ON tc.constraint_name = kcu.constraint_name
			  AND tc.table_schema = kcu.table_schema
			 WHERE tc.constraint_type = 'PRIMARY KEY'
			   AND tc.table_schema = current_schema() AND tc.table_name = ?
			 ORDER BY kcu.ordinal_position`, name,
		).Scan(&pk).Error; err != nil {
			return nil, err
		}
		t.PrimaryKey = pk

		type fkRow struct {
			Constraint string `gorm:"column:constraint_name"`
			FromCol    string `gorm:"column:from_col"`
			RefTable   string `gorm:"column:ref_table"`
			RefCol     string `gorm:"column:ref_col"`
		}
		var fkRows []fkRow
		if err := db.Raw(
			`SELECT tc.constraint_name,
			        kcu.column_name AS from_col,
			        ccu.table_name  AS ref_table,
			        ccu.column_name AS ref_col
			 FROM information_schema.table_constraints tc
			 JOIN information_schema.key_column_usage kcu
			   ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
			 JOIN information_schema.constraint_column_usage ccu
			   ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
			 WHERE tc.constraint_type = 'FOREIGN KEY'
			   AND tc.table_schema = current_schema() AND tc.table_name = ?
			 ORDER BY tc.constraint_name, kcu.ordinal_position`, name,
		).Scan(&fkRows).Error; err != nil {
			return nil, err
		}
		gfk := make([]genericFK, 0, len(fkRows))
		for _, r := range fkRows {
			gfk = append(gfk, genericFK{Constraint: r.Constraint, FromCol: r.FromCol, RefTable: r.RefTable, RefCol: r.RefCol})
		}
		t.ForeignKeys = groupForeignKeys(gfk)

		type idxRow struct {
			IndexName string `gorm:"column:index_name"`
			IsUnique  bool   `gorm:"column:is_unique"`
			Column    string `gorm:"column:column_name"`
		}
		var idxRows []idxRow
		if err := db.Raw(
			`SELECT i.relname AS index_name,
			        ix.indisunique AS is_unique,
			        a.attname AS column_name,
			        array_position(ix.indkey, a.attnum) AS ord
			 FROM pg_class t
			 JOIN pg_index ix ON t.oid = ix.indrelid
			 JOIN pg_class i ON i.oid = ix.indexrelid
			 JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
			 WHERE t.relkind = 'r' AND t.relname = ? AND NOT ix.indisprimary
			 ORDER BY i.relname, ord`, name,
		).Scan(&idxRows).Error; err != nil {
			return nil, err
		}
		gidx := make([]genericIdx, 0, len(idxRows))
		for _, r := range idxRows {
			gidx = append(gidx, genericIdx{Name: r.IndexName, Unique: r.IsUnique, Column: r.Column})
		}
		t.Indexes = groupIndexes(gidx)

		m.Tables = append(m.Tables, t)
	}

	sortTables(m.Tables)
	return m, nil
}

// ---------------- MySQL ----------------

func introspectMySQL(db *gorm.DB) (*Model, error) {
	m := &Model{Driver: "mysql"}

	var tableNames []string
	if err := db.Raw(
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
		 ORDER BY table_name`,
	).Scan(&tableNames).Error; err != nil {
		return nil, err
	}

	for _, name := range tableNames {
		t := Table{Name: name}

		type colRow struct {
			Name     string  `gorm:"column:column_name"`
			Type     string  `gorm:"column:column_type"`
			Nullable string  `gorm:"column:is_nullable"`
			Default  *string `gorm:"column:column_default"`
		}
		var cols []colRow
		if err := db.Raw(
			`SELECT column_name, column_type, is_nullable, column_default
			 FROM information_schema.columns
			 WHERE table_schema = DATABASE() AND table_name = ?
			 ORDER BY ordinal_position`, name,
		).Scan(&cols).Error; err != nil {
			return nil, err
		}
		for _, c := range cols {
			def := ""
			if c.Default != nil {
				def = *c.Default
			}
			t.Columns = append(t.Columns, Column{
				Name:     c.Name,
				Type:     c.Type,
				Nullable: strings.EqualFold(c.Nullable, "YES"),
				Default:  def,
			})
		}

		var pk []string
		if err := db.Raw(
			`SELECT column_name FROM information_schema.key_column_usage
			 WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = 'PRIMARY'
			 ORDER BY ordinal_position`, name,
		).Scan(&pk).Error; err != nil {
			return nil, err
		}
		t.PrimaryKey = pk

		type fkRow struct {
			Constraint string `gorm:"column:constraint_name"`
			FromCol    string `gorm:"column:from_col"`
			RefTable   string `gorm:"column:ref_table"`
			RefCol     string `gorm:"column:ref_col"`
		}
		var fkRows []fkRow
		if err := db.Raw(
			`SELECT constraint_name,
			        column_name AS from_col,
			        referenced_table_name AS ref_table,
			        referenced_column_name AS ref_col
			 FROM information_schema.key_column_usage
			 WHERE table_schema = DATABASE() AND table_name = ?
			   AND referenced_table_name IS NOT NULL
			 ORDER BY constraint_name, ordinal_position`, name,
		).Scan(&fkRows).Error; err != nil {
			return nil, err
		}
		gfk := make([]genericFK, 0, len(fkRows))
		for _, r := range fkRows {
			gfk = append(gfk, genericFK{Constraint: r.Constraint, FromCol: r.FromCol, RefTable: r.RefTable, RefCol: r.RefCol})
		}
		t.ForeignKeys = groupForeignKeys(gfk)

		type idxRow struct {
			IndexName string `gorm:"column:index_name"`
			NonUnique int    `gorm:"column:non_unique"`
			Column    string `gorm:"column:column_name"`
		}
		var idxRows []idxRow
		if err := db.Raw(
			`SELECT index_name, non_unique, column_name
			 FROM information_schema.statistics
			 WHERE table_schema = DATABASE() AND table_name = ? AND index_name != 'PRIMARY'
			 ORDER BY index_name, seq_in_index`, name,
		).Scan(&idxRows).Error; err != nil {
			return nil, err
		}
		generic := make([]genericIdx, 0, len(idxRows))
		for _, r := range idxRows {
			generic = append(generic, genericIdx{Name: r.IndexName, Unique: r.NonUnique == 0, Column: r.Column})
		}
		t.Indexes = groupIndexes(generic)

		m.Tables = append(m.Tables, t)
	}

	sortTables(m.Tables)
	return m, nil
}

// ---------------- shared grouping helpers ----------------

type genericFK struct {
	Constraint string
	FromCol    string
	RefTable   string
	RefCol     string
}

type genericIdx struct {
	Name   string
	Unique bool
	Column string
}

func groupForeignKeys(rows []genericFK) []ForeignKey {
	byName := map[string]*ForeignKey{}
	var order []string
	for _, r := range rows {
		fk, ok := byName[r.Constraint]
		if !ok {
			fk = &ForeignKey{RefTable: r.RefTable}
			byName[r.Constraint] = fk
			order = append(order, r.Constraint)
		}
		fk.Columns = append(fk.Columns, r.FromCol)
		fk.RefColumns = append(fk.RefColumns, r.RefCol)
	}
	var out []ForeignKey
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out
}

func groupIndexes(rows []genericIdx) []Index {
	byName := map[string]*Index{}
	var order []string
	for _, r := range rows {
		ix, ok := byName[r.Name]
		if !ok {
			ix = &Index{Name: r.Name, Unique: r.Unique}
			byName[r.Name] = ix
			order = append(order, r.Name)
		}
		ix.Columns = append(ix.Columns, r.Column)
	}
	var out []Index
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out
}
