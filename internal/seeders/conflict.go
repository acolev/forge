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

	// Собираем SQL: CREATE UNIQUE INDEX IF NOT EXISTS "name" ON "table" ("col",...)
	var b strings.Builder
	b.WriteString("CREATE UNIQUE INDEX IF NOT EXISTS ")
	b.WriteString(`"` + base + `" ON "` + table + `"` + " (")
	for i, c := range cols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(`"` + c + `"`)
	}
	b.WriteString(")")
	sql := b.String()

	// 1) Пытаемся с IF NOT EXISTS (PG ≥ 9.5)
	if err := tx.Exec(sql).Error; err == nil {
		return nil
	} else {
		// Если синтаксис не поддерживается (42601) — пробуем без IF NOT EXISTS и игнорируем дубликат имени.
		plain := strings.Replace(sql, " IF NOT EXISTS", "", 1)
		if err2 := tx.Exec(plain).Error; err2 != nil {
			// Если индекс уже есть (дубликат объекта) — просто игнорируем.
			// Коды возможных «дубликат» ошибок:
			// 42710 duplicate_object, 42P07 duplicate_table (для индексов тоже встречается)
			if isPgCode(err2, "42710") || isPgCode(err2, "42P07") {
				return nil
			}
			return fmt.Errorf("create unique index failed: %w", err2)
		}
		return nil
	}
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
