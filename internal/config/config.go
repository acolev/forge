package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	DefaultEnvFile      = ".env.forge"
	FallbackEnvFile     = ".env"
	DefaultPluginsDir   = ".forge/plugins"
	DefaultSQLiteDBPath = "database/database.db"
	ForgeDBDSNKey       = "FORGE_DB_DSN"
	ForgePluginsDirKey  = "FORGE_PLUGINS_DIR"
)

type Settings struct {
	EnvFile    string
	DBDSN      string
	PluginsDir string
}

func LoadEnv() error {
	values := map[string]string{}

	if err := mergeEnvFile(values, FallbackEnvFile); err != nil {
		return err
	}
	if err := mergeEnvFile(values, DefaultEnvFile); err != nil {
		return err
	}

	for key, value := range values {
		if current, exists := os.LookupEnv(key); exists && strings.TrimSpace(current) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s: %w", key, err)
		}
	}

	return nil
}

func CurrentSettings() (Settings, error) {
	if err := LoadEnv(); err != nil {
		return Settings{}, err
	}

	envFile := DefaultEnvFile
	if _, err := os.Stat(DefaultEnvFile); err != nil {
		if os.IsNotExist(err) {
			if _, fallbackErr := os.Stat(FallbackEnvFile); fallbackErr == nil {
				envFile = FallbackEnvFile
			}
		} else {
			return Settings{}, fmt.Errorf("stat %s: %w", DefaultEnvFile, err)
		}
	}

	dsn := strings.TrimSpace(os.Getenv(ForgeDBDSNKey))
	if dsn == "" {
		dsn = "sqlite://" + DefaultSQLiteDBPath
	}

	pluginsDir := strings.TrimSpace(os.Getenv(ForgePluginsDirKey))
	if pluginsDir == "" {
		pluginsDir = DefaultPluginsDir
	}

	return Settings{
		EnvFile:    envFile,
		DBDSN:      dsn,
		PluginsDir: pluginsDir,
	}, nil
}

func DefaultEnvLines() []string {
	return []string{
		ForgeDBDSNKey + "=sqlite://" + DefaultSQLiteDBPath,
		ForgePluginsDirKey + "=" + DefaultPluginsDir,
	}
}

func ResolvePluginsDir(projectDir string) (string, error) {
	settings, err := CurrentSettings()
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(settings.PluginsDir) {
		return settings.PluginsDir, nil
	}
	if projectDir == "" {
		return settings.PluginsDir, nil
	}
	return filepath.Join(projectDir, settings.PluginsDir), nil
}

func mergeEnvFile(dst map[string]string, filename string) error {
	values, err := godotenv.Read(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", filename, err)
	}
	for key, value := range values {
		dst[key] = value
	}
	return nil
}
