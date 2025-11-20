package seeders

import (
	"time"

	"gorm.io/gorm"
)

// Директория с YAML
const seedsDir = "./database/seeds"

const (
	defaultChunkSize  = 1000
	defaultBcryptCost = 12
)

// Учёт применённых сидов
type Seed struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"uniqueIndex;size:190"`
	Batch     int    `gorm:"index"`
	RanAt     time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// YAML формат (файл может быть списком seeds или одиночным сидом)
type YAMLConfig struct {
	Batch *int       `yaml:"batch,omitempty"`
	Seeds []YAMLSeed `yaml:"seeds,omitempty"`
}

type YAMLSeed struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // sql | fixture | go

	// sql
	SQL  string `yaml:"sql,omitempty"`
	File string `yaml:"file,omitempty"`

	// fixture
	Table       string           `yaml:"table,omitempty"`
	Rows        []map[string]any `yaml:"rows,omitempty"`
	OnConflict  string           `yaml:"on_conflict,omitempty"`  // "", "do_nothing", "update_all"
	ConflictKey []string         `yaml:"conflict_key,omitempty"` // для update_all
	ChunkSize   int              `yaml:"chunk_size,omitempty"`   // default 1000

	// bcrypt
	PasswordFields []string `yaml:"password_fields,omitempty"`
	PasswordCost   int      `yaml:"password_cost,omitempty"`

	// go
	Func string `yaml:"func,omitempty"`
}

// Для type:go
type GoSeederFunc func(db *gorm.DB) error

var goFuncs = map[string]GoSeederFunc{}

func RegisterFunc(name string, fn GoSeederFunc) { goFuncs[name] = fn }
