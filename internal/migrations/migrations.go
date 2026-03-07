package migrations

import (
	"context"
	"fmt"
	"forge/internal/hooks"
	"gorm.io/gorm"
	"io/ioutil"
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

	files, err := ioutil.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("unable to read migration directory: %v", err)
	}

	var lastBatch int = 0
	var migrationsToRun []os.FileInfo

	if err := db.Model(&Migration{}).Select("COALESCE(MAX(batch), 0)").Scan(&lastBatch).Error; err != nil {
		return fmt.Errorf("failed to get last batch: %v", err)
	}

	// Emit before-create event
	if err := hooks.Emit(ctx, hooks.Event{Name: "db.migrate.before", Payload: map[string]any{
		"lastBatch":       lastBatch,
		"db":              db,
		"migrationPath":   migrationPath,
		"files":           files,
		"migrationsToRun": migrationsToRun,
	}}); err != nil {
		return err
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

	if len(migrationsToRun) == 0 {
		fmt.Println("All migrations have already been applied.")
		return nil
	}

	// Sort the migration files by name in ascending order
	sort.Slice(migrationsToRun, func(i, j int) bool {
		return migrationsToRun[i].Name() < migrationsToRun[j].Name()
	})

	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, file := range migrationsToRun {
			content, err := ioutil.ReadFile(filepath.Join(migrationPath, file.Name()))
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

	// Emit before-create event
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

func RollbackLastMigration(db *gorm.DB) error {
	var lastBatch int
	if err := db.Model(&Migration{}).Select("MAX(batch)").Scan(&lastBatch).Error; err != nil {
		return fmt.Errorf("failed to get last batch: %v", err)
	}

	//if err := fireBeforeRollback(db); err != nil {
	//	return fmt.Errorf("before rollback hook failed: %w", err)
	//}

	var migrations []Migration
	if err := db.Where("batch = ?", lastBatch).Order("id DESC").Find(&migrations).Error; err != nil {
		return fmt.Errorf("failed to find migration records: %v", err)
	}

	for _, migration := range migrations {
		content, err := ioutil.ReadFile(filepath.Join(migrationPath, migration.FileName))
		if err != nil {
			return fmt.Errorf("unable to read file: %s, error: %v", migration.FileName, err)
		}

		sections := strings.Split(string(content), "-- DOWN")
		if len(sections) < 2 {
			return fmt.Errorf("no DOWN section found in migration: %s", migration.FileName)
		}

		downSection := strings.TrimPrefix(sections[1], "\n")
		fmt.Printf("Rolling back migration: %s\n", migration.FileName)
		if err := db.Exec(downSection).Error; err != nil {
			return fmt.Errorf("failed to execute rollback: %s, error: %v", migration.FileName, err)
		}

		if err := db.Delete(&migration).Error; err != nil {
			return fmt.Errorf("failed to delete migration record: %s, error: %v", migration.FileName, err)
		}
	}

	//if err := fireAfterRollback(db); err != nil {
	//	return fmt.Errorf("after rollback hook failed: %w", err)
	//}

	return nil
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
			if placeholder == name {
				placeholder = name
			}
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
