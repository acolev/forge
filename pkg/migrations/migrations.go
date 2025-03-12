package migrations

import (
	"fmt"
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
	ID        uint   `gorm:"primaryKey"`
	FileName  string `gorm:"unique"`
	Iteration int
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
	files, err := ioutil.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("unable to read migration directory: %v", err)
	}

	var lastIteration int
	var migrationsToRun []os.FileInfo

	if err := db.Model(&Migration{}).Select("MAX(iteration)").Scan(&lastIteration).Error; err != nil {
		fmt.Printf("Failed to get last iteration: %v\n", err)
		lastIteration = 0
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

	for _, file := range migrationsToRun {
		content, err := ioutil.ReadFile(filepath.Join(migrationPath, file.Name()))
		if err != nil {
			return fmt.Errorf("unable to read file: %s, error: %v", file.Name(), err)
		}

		sections := strings.Split(string(content), "-- DOWN")
		if len(sections) > 0 {
			upSection := strings.TrimPrefix(sections[0], "-- UP\n")
			fmt.Printf("Applying migration: %s\n", file.Name())
			if err := db.Exec(upSection).Error; err != nil {
				return fmt.Errorf("failed to execute migration: %s, error: %v", file.Name(), err)
			}

			migration := Migration{
				FileName:  file.Name(),
				Iteration: lastIteration + 1,
			}
			if err := db.Create(&migration).Error; err != nil {
				return fmt.Errorf("failed to record migration: %s, error: %v", file.Name(), err)
			}
		}
	}

	return nil
}
func RollbackLastMigration(db *gorm.DB) error {
	var lastIteration int
	if err := db.Model(&Migration{}).Select("MAX(iteration)").Scan(&lastIteration).Error; err != nil {
		return fmt.Errorf("failed to get last iteration: %v", err)
	}

	var migrations []Migration
	if err := db.Where("iteration = ?", lastIteration).Find(&migrations).Error; err != nil {
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

	return nil
}

func createMigrationFile(path string, tableName string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file: %s, error: %v", path, err)
	}
	defer f.Close()

	stubName := strings.Join(strings.Split(tableName, "_")[:len(strings.Split(tableName, "_"))-1], "_")
	tpl, err := getTemplate(stubName)
	if err != nil {
		tpl = ""
	} else {
		// Remove the prefix from the table name
		prefix := stubName + "_"
		tableNameWithoutPrefix := strings.TrimPrefix(tableName, prefix)
		tpl = strings.Replace(tpl, "{table_name}", tableNameWithoutPrefix, -1)
	}

	if _, err := f.WriteString(tpl); err != nil {
		return fmt.Errorf("unable to write to the file: %s, error: %v", path, err)
	}
	return nil
}

func ensureMigrationDirectory() error {
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		if err := os.MkdirAll(migrationPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}
	return nil
}
