package config

import (
	"fmt"
	"net/url"
	"strings"
)

// DSNParts holds the decomposed pieces of a database DSN, used to pre-fill the
// configuration wizard.
type DSNParts struct {
	Driver     string // sqlite | postgres | mysql
	Host       string
	Port       string
	User       string
	Password   string
	DBName     string
	SQLitePath string
}

// DefaultPort returns the conventional port for a driver.
func DefaultPort(driver string) string {
	switch driver {
	case "postgres":
		return "5432"
	case "mysql":
		return "3306"
	default:
		return ""
	}
}

// BuildDSN assembles a Forge DSN string from individual parts.
func BuildDSN(p DSNParts) (string, error) {
	switch p.Driver {
	case "sqlite":
		path := strings.TrimSpace(p.SQLitePath)
		if path == "" {
			path = DefaultSQLiteDBPath
		}
		return "sqlite://" + path, nil

	case "postgres", "mysql":
		host := strings.TrimSpace(p.Host)
		if host == "" {
			host = "localhost"
		}
		port := strings.TrimSpace(p.Port)
		if port == "" {
			port = DefaultPort(p.Driver)
		}
		if strings.TrimSpace(p.User) == "" {
			return "", fmt.Errorf("%s requires a user", p.Driver)
		}
		if strings.TrimSpace(p.DBName) == "" {
			return "", fmt.Errorf("%s requires a database name", p.Driver)
		}
		u := url.URL{
			Scheme: p.Driver,
			User:   url.UserPassword(p.User, p.Password),
			Host:   host + ":" + port,
			Path:   "/" + strings.TrimPrefix(p.DBName, "/"),
		}
		return u.String(), nil

	default:
		return "", fmt.Errorf("unsupported driver %q (use: sqlite, postgres, mysql)", p.Driver)
	}
}

// ParseDSN decomposes an existing DSN for pre-filling the wizard. It is
// best-effort: unrecognized input yields a sqlite default.
func ParseDSN(dsn string) DSNParts {
	dsn = strings.TrimSpace(dsn)
	lower := strings.ToLower(dsn)

	switch {
	case strings.HasPrefix(lower, "sqlite://"):
		return DSNParts{Driver: "sqlite", SQLitePath: dsn[len("sqlite://"):]}

	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"),
		strings.HasPrefix(lower, "mysql://"):
		u, err := url.Parse(dsn)
		if err != nil {
			return DSNParts{Driver: "sqlite", SQLitePath: DefaultSQLiteDBPath}
		}
		driver := "postgres"
		if strings.HasPrefix(lower, "mysql://") {
			driver = "mysql"
		}
		p := DSNParts{
			Driver: driver,
			Host:   u.Hostname(),
			Port:   u.Port(),
			DBName: strings.TrimPrefix(u.Path, "/"),
		}
		if u.User != nil {
			p.User = u.User.Username()
			p.Password, _ = u.User.Password()
		}
		if p.Port == "" {
			p.Port = DefaultPort(driver)
		}
		return p

	default:
		return DSNParts{Driver: "sqlite", SQLitePath: DefaultSQLiteDBPath}
	}
}
