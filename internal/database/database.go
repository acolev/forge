package database

import (
	"fmt"
	"forge/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"strings"
)

var DB *gorm.DB

type Migration struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	FileName string `json:"fileName" gorm:"unique"`
	Batch    int    `json:"batch"`
}

// Connect opens a database connection from a Forge DSN without running any
// migrations. Used by the config wizard to test connectivity.
func Connect(forgeDSN string) (*gorm.DB, error) {
	driver, dsn, err := parseForgeDBDSN(forgeDSN)
	if err != nil {
		return nil, err
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	var db *gorm.DB
	switch driver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), gormConfig)
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "postgres":
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", driver)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	return db, nil
}

func InitDB() (*gorm.DB, error) {
	settings, err := config.CurrentSettings()
	if err != nil {
		return nil, err
	}

	DB, err = Connect(settings.DBDSN)
	if err != nil {
		return nil, err
	}

	// AutoMigrate the Migration struct to create the migrations table if it doesn't exist
	//upSection := "CREATE TABLE migrations (\n    id INTEGER PRIMARY KEY AUTO_INCREMENT,\n    file_name TEXT,\n    iteration INTEGER,\n    UNIQUE KEY uni_migrations_file_name (file_name(255))\n);"
	//if err := DB.Exec(upSection).Error; err != nil {
	//	return nil, fmt.Errorf("failed to create migrations table: %v", err)
	//}
	if err := DB.AutoMigrate(&Migration{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return DB, nil
}

func parseForgeDBDSN(raw string) (string, string, error) {
	dsn := strings.TrimSpace(raw)
	if dsn == "" {
		return "", "", fmt.Errorf("FORGE_DB_DSN cannot be empty")
	}

	lower := strings.ToLower(dsn)

	switch {
	case strings.HasPrefix(lower, "sqlite://"):
		sqliteDSN := dsn[len("sqlite://"):]
		if strings.TrimSpace(sqliteDSN) == "" {
			return "", "", fmt.Errorf("FORGE_DB_DSN sqlite path cannot be empty")
		}
		return "sqlite", sqliteDSN, nil
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
		return "postgres", dsn, nil
	case strings.HasPrefix(lower, "mysql://"):
		mysqlDSN, err := mysqlURLToDSN(dsn)
		if err != nil {
			return "", "", err
		}
		return "mysql", mysqlDSN, nil
	default:
		return "sqlite", dsn, nil
	}
}

func mysqlURLToDSN(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid mysql FORGE_DB_DSN: %w", err)
	}

	host := strings.TrimSpace(u.Host)
	dbName := strings.TrimPrefix(u.EscapedPath(), "/")
	if host == "" || dbName == "" {
		return "", fmt.Errorf("mysql FORGE_DB_DSN must include host and database name")
	}

	user := ""
	password := ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}
	if user == "" {
		return "", fmt.Errorf("mysql FORGE_DB_DSN must include username")
	}

	query := u.Query()
	if query.Get("charset") == "" {
		query.Set("charset", "utf8mb4")
	}
	if query.Get("parseTime") == "" {
		query.Set("parseTime", "True")
	}
	if query.Get("loc") == "" {
		query.Set("loc", "Local")
	}

	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", user, password, host, dbName, query.Encode()), nil
}
