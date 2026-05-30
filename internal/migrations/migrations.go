package migrations

import (
	"context"
	"fmt"
	"forge/internal/hooks"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const migrationPath = "./database/migrations"

type Migration struct {
	ID       uint   `gorm:"primaryKey"`
	FileName string `gorm:"unique"`
	Batch    int
}

func CreateMigration(tableName string) error {
	if err := ensureMigrationDirectory(); err != nil {
		return err
	}

	unixTime := time.Now().Unix()
	migrationName := fmt.Sprintf("%d_%s", unixTime, tableName)
	migrationFilePath := filepath.Join(migrationPath, migrationName+".sql")
	if err := createMigrationFile(migrationFilePath, tableName); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", migrationFilePath)
	return nil
}

func RunMigrations(db *gorm.DB) error {
	ctx := context.Background()
	startedAt := time.Now()

	// Ensure the bookkeeping table exists — RunMigrations may be called after
	// `db fresh` has dropped every table, including this one.
	if err := db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %v", err)
	}

	files, err := os.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("unable to read migration directory: %v", err)
	}

	var lastBatch int = 0
	var migrationsToRun []os.DirEntry

	if err := db.Model(&Migration{}).Select("COALESCE(MAX(batch), 0)").Scan(&lastBatch).Error; err != nil {
		return fmt.Errorf("failed to get last batch: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			var migration Migration
			if err := db.Where("file_name = ?", file.Name()).First(&migration).Error; err == nil {
				continue // Migration already applied
			}
			migrationsToRun = append(migrationsToRun, file)
		}
	}

	// Sort the migration files by name in ascending order
	sort.Slice(migrationsToRun, func(i, j int) bool {
		return migrationsToRun[i].Name() < migrationsToRun[j].Name()
	})

	// Emit before-migrate event with the actual (computed) list of pending migrations.
	if err := hooks.Emit(ctx, hooks.Event{Name: "db.migrate.before", Payload: map[string]any{
		"lastBatch":       lastBatch,
		"db":              db,
		"migrationPath":   migrationPath,
		"files":           files,
		"migrationsToRun": migrationsToRun,
	}}); err != nil {
		return err
	}

	if len(migrationsToRun) == 0 {
		fmt.Println("All migrations have already been applied.")
		return nil
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, file := range migrationsToRun {
			content, err := os.ReadFile(filepath.Join(migrationPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to read file: %s, error: %v", file.Name(), err)
			}

			sections := strings.Split(string(content), "-- DOWN")
			if len(sections) == 0 {
				return fmt.Errorf("invalid migration format: %s", file.Name())
			}

			upSection := strings.TrimPrefix(sections[0], "-- UP\n")
			fmt.Printf("Applying migration: %s\n", file.Name())
			if err := tx.Exec(upSection).Error; err != nil {
				return fmt.Errorf("failed to execute migration: %s, error: %v", file.Name(), err)
			}

			migration := Migration{
				FileName: file.Name(),
				Batch:    lastBatch + 1,
			}
			if err := tx.Create(&migration).Error; err != nil {
				return fmt.Errorf("failed to record migration: %s, error: %v", file.Name(), err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Emit after-migrate event
	if err := hooks.Emit(ctx, hooks.Event{Name: "db.migrate.after", Payload: map[string]any{
		"lastBatch":      lastBatch,
		"db":             db,
		"migrationPath":  migrationPath,
		"files":          files,
		"migrationsRun":  migrationsToRun,
		"newBatchNumber": lastBatch + 1,
		"appliedAt":      time.Now(),
		"duration":       time.Since(startedAt),
		"error":          nil,
	}}); err != nil {
		return err
	}

	return nil
}

// RollbackLastMigration rolls back the most recent batch.
func RollbackLastMigration(db *gorm.DB) error {
	return RollbackBatches(db, 1)
}

// RollbackBatches rolls back the last `steps` batches (most recent first).
func RollbackBatches(db *gorm.DB, steps int) error {
	if steps <= 0 {
		steps = 1
	}
	for s := 0; s < steps; s++ {
		batch, err := currentBatch(db)
		if err != nil {
			return err
		}
		if batch == 0 {
			if s == 0 {
				fmt.Println("No migrations to roll back.")
			}
			return nil
		}
		if err := rollbackBatch(db, batch); err != nil {
			return err
		}
	}
	return nil
}

// ResetMigrations rolls back every applied batch.
func ResetMigrations(db *gorm.DB) error {
	for {
		batch, err := currentBatch(db)
		if err != nil {
			return err
		}
		if batch == 0 {
			return nil
		}
		if err := rollbackBatch(db, batch); err != nil {
			return err
		}
	}
}

func currentBatch(db *gorm.DB) (int, error) {
	var batch int
	if err := db.Model(&Migration{}).Select("COALESCE(MAX(batch), 0)").Scan(&batch).Error; err != nil {
		return 0, fmt.Errorf("failed to get last batch: %v", err)
	}
	return batch, nil
}

func rollbackBatch(db *gorm.DB, batch int) error {
	var migrations []Migration
	if err := db.Where("batch = ?", batch).Order("id DESC").Find(&migrations).Error; err != nil {
		return fmt.Errorf("failed to find migration records: %v", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, migration := range migrations {
			content, err := os.ReadFile(filepath.Join(migrationPath, migration.FileName))
			if err != nil {
				return fmt.Errorf("unable to read file: %s, error: %v", migration.FileName, err)
			}

			sections := strings.Split(string(content), "-- DOWN")
			if len(sections) < 2 {
				return fmt.Errorf("no DOWN section found in migration: %s", migration.FileName)
			}

			downSection := strings.TrimPrefix(sections[1], "\n")
			fmt.Printf("Rolling back migration: %s\n", migration.FileName)
			if err := tx.Exec(downSection).Error; err != nil {
				return fmt.Errorf("failed to execute rollback: %s, error: %v", migration.FileName, err)
			}

			if err := tx.Delete(&migration).Error; err != nil {
				return fmt.Errorf("failed to delete migration record: %s, error: %v", migration.FileName, err)
			}
		}
		return nil
	})
}

// StatusRow describes one migration file and whether it has been applied.
type StatusRow struct {
	FileName string
	Applied  bool
	Batch    int
}

// GetStatus returns the apply-state of every migration file on disk.
func GetStatus(db *gorm.DB) ([]StatusRow, error) {
	files, err := os.ReadDir(migrationPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read migration directory: %v", err)
	}

	var applied []Migration
	if err := db.Find(&applied).Error; err != nil {
		return nil, fmt.Errorf("failed to read applied migrations: %v", err)
	}
	byName := map[string]Migration{}
	for _, a := range applied {
		byName[a.FileName] = a
	}

	var rows []StatusRow
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		row := StatusRow{FileName: f.Name()}
		if a, ok := byName[f.Name()]; ok {
			row.Applied = true
			row.Batch = a.Batch
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].FileName < rows[j].FileName })
	return rows, nil
}

// PendingUpSQL returns the UP SQL of every not-yet-applied migration, in order.
// Used by `migrate --dry-run`.
func PendingUpSQL(db *gorm.DB) ([]string, []string, error) {
	rows, err := GetStatus(db)
	if err != nil {
		return nil, nil, err
	}
	var names, sqls []string
	for _, r := range rows {
		if r.Applied {
			continue
		}
		content, err := os.ReadFile(filepath.Join(migrationPath, r.FileName))
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read file: %s, error: %v", r.FileName, err)
		}
		up := strings.TrimPrefix(strings.Split(string(content), "-- DOWN")[0], "-- UP\n")
		names = append(names, r.FileName)
		sqls = append(sqls, strings.TrimSpace(up))
	}
	return names, sqls, nil
}

func createMigrationFile(path string, tableName string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file: %s, error: %v", path, err)
	}
	defer f.Close()

	stubName, placeholderName := resolveStubAndPlaceholder(tableName)
	tpl, err := getTemplate(stubName)
	if err != nil {
		tpl = "" +
			"-- UP\n" +
			"\n" +
			"-- DOWN\n"
	} else {
		tpl = strings.Replace(tpl, "{table_name}", placeholderName, -1)
	}

	if _, err := f.WriteString(tpl); err != nil {
		return fmt.Errorf("unable to write to the file: %s, error: %v", path, err)
	}
	return nil
}

func resolveStubAndPlaceholder(name string) (string, string) {
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return "", name
	}

	for i := len(parts); i > 0; i-- {
		candidate := strings.Join(parts[:i], "_")
		if _, err := getTemplate(candidate); err == nil {
			placeholder := strings.TrimPrefix(name, candidate+"_")
			return candidate, placeholder
		}
	}

	switch parts[0] {
	case "create":
		return "create_table", strings.TrimPrefix(name, "create_")
	case "update":
		return "update_table", strings.TrimPrefix(name, "update_")
	}

	return "", name
}

func ensureMigrationDirectory() error {
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		if err := os.MkdirAll(migrationPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}
	return nil
}
