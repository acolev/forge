package migrations

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const migrationPath = "./database/migrations"

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

func RunMigrations() error {
	files, err := ioutil.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("unable to read migration directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			content, err := ioutil.ReadFile(filepath.Join(migrationPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to read file: %s, error: %v", file.Name(), err)
			}

			sections := strings.Split(string(content), "-- DOWN")
			if len(sections) > 0 {
				upSection := strings.TrimPrefix(sections[0], "-- UP\n")
				fmt.Printf("File: %s\n%s\n", file.Name(), upSection)
			}
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
