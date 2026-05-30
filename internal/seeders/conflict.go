package seeders

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Создаёт UNIQUE INDEX на (cols), если его ещё нет.
// Без сложных CTE — просто пытаемся создать и игнорируем «уже существует».
func ensureUniqueForConflict(tx *gorm.DB, table string, cols []string) error {
	if len(cols) == 0 {
		return nil
	}

	// Дет. имя индекса: ux_<table>_<col1>_<col2>...
	base := "ux_" + table + "_" + strings.Join(cols, "_")
	if len(base) > 63 {
		sum := fmt.Sprintf("%x", sha1.Sum([]byte(base)))
		prefix := base
		if len(prefix) > 40 {
			prefix = prefix[:40]
		}
		base = prefix + "_" + sum[:16]
	}

	q := identQuoter(tx.Dialector.Name())
	var cb strings.Builder
	for i, c := range cols {
		if i > 0 {
			cb.WriteString(", ")
		}
		cb.WriteString(q(c))
	}
	colList := cb.String()

	switch tx.Dialector.Name() {
	case "mysql":
		// MySQL не поддерживает CREATE INDEX IF NOT EXISTS — создаём и глотаем дубликат (1061).
		sql := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", q(base), q(table), colList)
		if err := tx.Exec(sql).Error; err != nil {
			if isMySQLDuplicateIndex(err) {
				return nil
			}
			return fmt.Errorf("create unique index failed: %w", err)
		}
		return nil
	default:
		// postgres / sqlite поддерживают IF NOT EXISTS.
		sql := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)", q(base), q(table), colList)
		if err := tx.Exec(sql).Error; err == nil {
			return nil
		}
		// Фолбэк без IF NOT EXISTS — игнорируем дубликат объекта.
		plain := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", q(base), q(table), colList)
		if err := tx.Exec(plain).Error; err != nil {
			// 42710 duplicate_object, 42P07 duplicate_table
			if isPgCode(err, "42710") || isPgCode(err, "42P07") {
				return nil
			}
			return fmt.Errorf("create unique index failed: %w", err)
		}
		return nil
	}
}

// identQuoter возвращает функцию экранирования идентификаторов под диалект.
func identQuoter(dialect string) func(string) string {
	if dialect == "mysql" {
		return func(s string) string { return "`" + strings.ReplaceAll(s, "`", "``") + "`" }
	}
	return func(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
}

// isMySQLDuplicateIndex распознаёт MySQL error 1061 (Duplicate key name).
func isMySQLDuplicateIndex(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "1061") || strings.Contains(msg, "duplicate key name")
}

// Выдёргиваем SQLSTATE из ошибок GORM/pgx, чтобы сравнить код.
func isPgCode(err error, code string) bool {
	type causer interface{ Unwrap() error }
	type coder interface{ SQLState() string }

	// раскручиваем вложенные ошибки
	e := err
	for e != nil {
		if c, ok := e.(coder); ok {
			return c.SQLState() == code
		}
		u, ok := e.(causer)
		if !ok {
			break
		}
		e = u.Unwrap()
	}
	// у некоторых драйверов код в тексте — сделаем грубую проверку
	return strings.Contains(strings.ToUpper(err.Error()), code)
}
