package seeders

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ensureTable(db *gorm.DB) error { return db.AutoMigrate(&Seed{}) }

func executedMap(db *gorm.DB) (map[string]bool, int, error) {
	var rows []Seed
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	m := make(map[string]bool, len(rows))
	last := 0
	for _, r := range rows {
		m[r.Name] = true
		if r.Batch > last {
			last = r.Batch
		}
	}
	return m, last, nil
}

// API
func ApplyAll(db *gorm.DB) error {
	if err := ensureTable(db); err != nil {
		return err
	}

	files, err := listYAML(seedsDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No seed yaml files found.")
		return nil
	}

	done, last, err := executedMap(db)
	if err != nil {
		return err
	}
	defaultBatch := last + 1

	for _, path := range files {
		cfg, single, err := loadConfig(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		base := filepath.Base(path)

		fileBatch := defaultBatch
		if cfg != nil && cfg.Batch != nil {
			fileBatch = *cfg.Batch
		}

		var seeds []YAMLSeed
		switch {
		case cfg != nil && len(cfg.Seeds) > 0:
			seeds = cfg.Seeds
		case single != nil:
			seeds = []YAMLSeed{*single}
		default:
			continue
		}

		for i := range seeds {
			if strings.TrimSpace(seeds[i].Name) == "" {
				seeds[i].Name = fmt.Sprintf("%s#%03d", base, i+1)
			}
		}

		for _, s := range seeds {
			if done[s.Name] {
				continue
			}
			if err := runSeed(db, filepath.Dir(path), s, fileBatch); err != nil {
				return fmt.Errorf("seed %q (%s): %w", s.Name, base, err)
			}
		}
	}
	fmt.Println("YAML seeds applied.")
	return nil
}

func ApplyOnly(db *gorm.DB, names []string) error {
	if err := ensureTable(db); err != nil {
		return err
	}
	files, err := listYAML(seedsDir)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, n := range names {
		want[strings.TrimSpace(n)] = true
	}
	done, last, err := executedMap(db)
	if err != nil {
		return err
	}
	batch := last + 1

	for _, path := range files {
		cfg, single, err := loadConfig(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		base := filepath.Base(path)

		var seeds []YAMLSeed
		switch {
		case cfg != nil && len(cfg.Seeds) > 0:
			seeds = cfg.Seeds
		case single != nil:
			seeds = []YAMLSeed{*single}
		default:
			continue
		}
		for i := range seeds {
			if strings.TrimSpace(seeds[i].Name) == "" {
				seeds[i].Name = fmt.Sprintf("%s#%03d", base, i+1)
			}
		}
		for _, s := range seeds {
			if !want[s.Name] || done[s.Name] {
				continue
			}
			useBatch := batch
			if cfg != nil && cfg.Batch != nil {
				useBatch = *cfg.Batch
			}
			if err := runSeed(db, filepath.Dir(path), s, useBatch); err != nil {
				return fmt.Errorf("seed %q (%s): %w", s.Name, base, err)
			}
		}
	}
	return nil
}

func Status(db *gorm.DB) error {
	if err := ensureTable(db); err != nil {
		return err
	}
	var rows []Seed
	if err := db.Order("batch asc, name asc").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println("No seeders executed.")
		return nil
	}
	for _, r := range rows {
		fmt.Printf("[%d] %s at %s\n", r.Batch, r.Name, r.RanAt.Format(time.RFC3339))
	}
	return nil
}

func Reset(db *gorm.DB) error {
	if err := ensureTable(db); err != nil {
		return err
	}
	return db.Exec("DELETE FROM seeds").Error
}

// загрузка YAML
func listYAML(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".yaml") || strings.HasSuffix(n, ".yml") {
			files = append(files, filepath.Join(dir, n))
		}
	}
	sort.Strings(files)
	return files, nil
}

func loadConfig(path string) (*YAMLConfig, *YAMLSeed, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var cfg YAMLConfig
	if err := yaml.Unmarshal(b, &cfg); err == nil && (cfg.Batch != nil || len(cfg.Seeds) > 0) {
		return &cfg, nil, nil
	}
	var single YAMLSeed
	if err := yaml.Unmarshal(b, &single); err == nil && (single.Type != "" || single.SQL != "" || single.Table != "" || single.Func != "") {
		return nil, &single, nil
	}
	return nil, nil, errors.New("invalid yaml seed file format")
}

// исполнение
func runSeed(db *gorm.DB, baseDir string, s YAMLSeed, batch int) error {
	switch strings.ToLower(strings.TrimSpace(s.Type)) {
	case "sql":
		return runSQL(db, baseDir, s, batch)
	case "fixture":
		return runFixture(db, s, batch)
	case "go":
		return runGo(db, s, batch)
	default:
		return fmt.Errorf("unknown type %q", s.Type)
	}
}

func runSQL(db *gorm.DB, baseDir string, s YAMLSeed, batch int) error {
	sqlText := strings.TrimSpace(s.SQL)
	if s.File != "" {
		full := s.File
		if !filepath.IsAbs(full) {
			full = filepath.Join(baseDir, s.File)
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		sqlText = string(raw)
	}
	if sqlText == "" {
		return errors.New("sql seed: empty SQL")
	}

	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Exec(sqlText).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&Seed{Name: s.Name, Batch: batch, RanAt: time.Now()}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func runFixture(db *gorm.DB, s YAMLSeed, batch int) error {
	if s.Table == "" {
		return errors.New("fixture seed: table is required")
	}
	if len(s.Rows) == 0 {
		return db.Create(&Seed{Name: s.Name, Batch: batch, RanAt: time.Now()}).Error
	}

	chunk := s.ChunkSize
	if chunk <= 0 {
		chunk = defaultChunkSize
	}

	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 1) $ref (включая вложенные в where)
	for i := range s.Rows {
		if err := resolveRowRefs(tx, s.Rows[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("resolve $ref failed: %w", err)
		}
	}

	// 2) bcrypt
	fields := s.PasswordFields
	if len(fields) == 0 {
		fields = []string{"password"}
	}
	cost := s.PasswordCost
	if cost <= 0 {
		cost = defaultBcryptCost
	}
	for i := range s.Rows {
		if err := hashPasswordFieldsIfPresent(s.Rows[i], fields, cost); err != nil {
			tx.Rollback()
			return fmt.Errorf("hash password failed: %w", err)
		}
	}

	// 3) нормализация под типы колонок
	colKinds, err := pgColumnKinds(tx, s.Table)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("detect column kinds failed: %w", err)
	}
	for i := range s.Rows {
		normalizeRowJSONForTable(s.Rows[i], colKinds)
		normalizeRowByteaForTable(s.Rows[i], colKinds)
	}

	// 4) вставка / апсерт (+ авто-индекс при update_all)
	switch strings.ToLower(s.OnConflict) {
	case "do_nothing":
		for i := 0; i < len(s.Rows); i += chunk {
			end := min(endIndex(i, chunk, len(s.Rows)), len(s.Rows))
			if err := tx.Table(s.Table).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(s.Rows[i:end]).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	case "update_all":
		if len(s.ConflictKey) == 0 {
			tx.Rollback()
			return errors.New("fixture seed: conflict_key is required for update_all")
		}
		// авто-UNIQUE-индекс (если нет)
		if err := ensureUniqueForConflict(tx, s.Table, s.ConflictKey); err != nil {
			tx.Rollback()
			return fmt.Errorf("ensure unique index for on_conflict failed: %w", err)
		}

		cols := make([]clause.Column, 0, len(s.ConflictKey))
		for _, k := range s.ConflictKey {
			cols = append(cols, clause.Column{Name: k})
		}

		for i := 0; i < len(s.Rows); i += chunk {
			end := min(endIndex(i, chunk, len(s.Rows)), len(s.Rows))
			update := map[string]any{}
			for k := range s.Rows[i] {
				if !strIn(s.ConflictKey, k) {
					update[k] = clause.Expr{SQL: "EXCLUDED." + k}
				}
			}
			if len(update) == 0 {
				update["updated_at"] = clause.Expr{SQL: "NOW()"}
			}
			if err := tx.Table(s.Table).
				Clauses(clause.OnConflict{Columns: cols, DoUpdates: clause.Assignments(update)}).
				Create(s.Rows[i:end]).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	default:
		for i := 0; i < len(s.Rows); i += chunk {
			end := min(endIndex(i, chunk, len(s.Rows)), len(s.Rows))
			if err := tx.Table(s.Table).Create(s.Rows[i:end]).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if err := tx.Create(&Seed{Name: s.Name, Batch: batch, RanAt: time.Now()}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func runGo(db *gorm.DB, s YAMLSeed, batch int) error {
	fn := goFuncs[s.Func]
	if fn == nil {
		return fmt.Errorf("go seed: function %q is not registered", s.Func)
	}
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&Seed{Name: s.Name, Batch: batch, RanAt: time.Now()}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// helpers
func endIndex(i, chunk, total int) int {
	e := i + chunk
	if e > total {
		e = total
	}
	return e
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func strIn(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
