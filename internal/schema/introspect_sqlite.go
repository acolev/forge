package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func introspectSQLite(db *gorm.DB) (*Model, error) {
	m := &Model{Driver: "sqlite"}

	var tableNames []string
	if err := db.Raw(
		`SELECT name FROM sqlite_master
		 WHERE type='table' AND name NOT LIKE 'sqlite_%'
		 ORDER BY name`,
	).Scan(&tableNames).Error; err != nil {
		return nil, err
	}

	for _, name := range tableNames {
		t := Table{Name: name}

		// Columns + primary key (PRAGMA does not accept bind params).
		type colInfo struct {
			Cid     int    `gorm:"column:cid"`
			Name    string `gorm:"column:name"`
			Type    string `gorm:"column:type"`
			NotNull int    `gorm:"column:notnull"`
			Dflt    *string `gorm:"column:dflt_value"`
			PK      int    `gorm:"column:pk"`
		}
		var cols []colInfo
		if err := db.Raw(fmt.Sprintf("PRAGMA table_info(%q)", name)).Scan(&cols).Error; err != nil {
			return nil, fmt.Errorf("table_info(%s): %w", name, err)
		}

		// PK columns ordered by their pk index.
		type pkCol struct {
			name string
			ord  int
		}
		var pks []pkCol
		for _, c := range cols {
			def := ""
			if c.Dflt != nil {
				def = *c.Dflt
			}
			t.Columns = append(t.Columns, Column{
				Name:     c.Name,
				Type:     strings.TrimSpace(c.Type),
				Nullable: c.NotNull == 0 && c.PK == 0,
				Default:  def,
			})
			if c.PK > 0 {
				pks = append(pks, pkCol{c.Name, c.PK})
			}
		}
		for i := 0; i < len(pks); i++ {
			for j := i + 1; j < len(pks); j++ {
				if pks[j].ord < pks[i].ord {
					pks[i], pks[j] = pks[j], pks[i]
				}
			}
		}
		for _, p := range pks {
			t.PrimaryKey = append(t.PrimaryKey, p.name)
		}

		// Foreign keys.
		type fkInfo struct {
			ID    int    `gorm:"column:id"`
			Seq   int    `gorm:"column:seq"`
			Table string `gorm:"column:table"`
			From  string `gorm:"column:from"`
			To    string `gorm:"column:to"`
		}
		var fks []fkInfo
		if err := db.Raw(fmt.Sprintf("PRAGMA foreign_key_list(%q)", name)).Scan(&fks).Error; err != nil {
			return nil, fmt.Errorf("foreign_key_list(%s): %w", name, err)
		}
		fkByID := map[int]*ForeignKey{}
		var fkOrder []int
		for _, fk := range fks {
			cur, ok := fkByID[fk.ID]
			if !ok {
				cur = &ForeignKey{RefTable: fk.Table}
				fkByID[fk.ID] = cur
				fkOrder = append(fkOrder, fk.ID)
			}
			cur.Columns = append(cur.Columns, fk.From)
			cur.RefColumns = append(cur.RefColumns, fk.To)
		}
		for _, id := range fkOrder {
			t.ForeignKeys = append(t.ForeignKeys, *fkByID[id])
		}

		// Indexes (skip auto-indexes created for UNIQUE/PK constraints).
		type idxInfo struct {
			Seq    int    `gorm:"column:seq"`
			Name   string `gorm:"column:name"`
			Unique int    `gorm:"column:unique"`
			Origin string `gorm:"column:origin"`
		}
		var idxs []idxInfo
		if err := db.Raw(fmt.Sprintf("PRAGMA index_list(%q)", name)).Scan(&idxs).Error; err != nil {
			return nil, fmt.Errorf("index_list(%s): %w", name, err)
		}
		for _, ix := range idxs {
			if strings.HasPrefix(ix.Name, "sqlite_autoindex_") {
				continue
			}
			type idxCol struct {
				Seqno int    `gorm:"column:seqno"`
				Cid   int    `gorm:"column:cid"`
				Name  string `gorm:"column:name"`
			}
			var icols []idxCol
			if err := db.Raw(fmt.Sprintf("PRAGMA index_info(%q)", ix.Name)).Scan(&icols).Error; err != nil {
				return nil, fmt.Errorf("index_info(%s): %w", ix.Name, err)
			}
			idx := Index{Name: ix.Name, Unique: ix.Unique == 1}
			for _, ic := range icols {
				idx.Columns = append(idx.Columns, ic.Name)
			}
			t.Indexes = append(t.Indexes, idx)
		}

		m.Tables = append(m.Tables, t)
	}

	sortTables(m.Tables)
	return m, nil
}
