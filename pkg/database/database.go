package database

import (
	"fmt"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
)

var DB *gorm.DB

type Migration struct {
	ID        uint   `gorm:"primaryKey"`
	FileName  string `gorm:"unique"`
	Iteration int
}

func InitDB() (*gorm.DB, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	// Get database configuration from environment variables
	dbDriver := os.Getenv("DB_DRIVER")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	source := "database/database.db"

	var err error

	// Set GORM logger to Silent mode
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch dbDriver {
	case "sqlite":
		DB, err = gorm.Open(sqlite.Open(source), gormConfig)
	case "mysql":
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)
		DB, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)
		DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// AutoMigrate the Migration struct to create the migrations table if it doesn't exist
	if err := DB.AutoMigrate(&Migration{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return DB, nil
}
