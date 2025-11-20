package seeders

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// типы колонок (json/jsonb/bytea)
func pgColumnKinds(tx *gorm.DB, table string) (map[string]string, error) {
	type rec struct{ Col, Kind string }
	var rows []rec
	const q = `
SELECT c.column_name AS col, c.udt_name AS kind
FROM information_schema.columns c
WHERE c.table_name = ? AND c.table_schema = current_schema()
`
	if err := tx.Raw(q, table).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		k := strings.ToLower(r.Kind)
		switch k {
		case "json", "jsonb", "bytea":
			out[r.Col] = k
		}
	}
	return out, nil
}

// json/jsonb: любое значение → валидный JSON + явный каст
func normalizeRowJSONForTable(row map[string]any, colKinds map[string]string) {
	for k, v := range row {
		kind, ok := colKinds[k]
		if !ok || (kind != "json" && kind != "jsonb") {
			continue
		}
		if _, isExpr := v.(clause.Expr); isExpr {
			continue
		}
		b, _ := json.Marshal(v)
		if kind == "jsonb" {
			row[k] = gorm.Expr("?::jsonb", string(b))
		} else {
			row[k] = gorm.Expr("?::json", string(b))
		}
	}
}

// bytea: map/array → JSON bytes; "base64:..." / "hex:..." → декод; иначе []byte(s)
func normalizeRowByteaForTable(row map[string]any, colKinds map[string]string) {
	for k, v := range row {
		if colKinds[k] != "bytea" {
			continue
		}
		switch vv := v.(type) {
		case []byte:
			continue
		case map[string]any, []any:
			b, _ := json.Marshal(vv)
			row[k] = b
		case string:
			s := strings.TrimSpace(vv)
			if strings.HasPrefix(s, "base64:") {
				raw := strings.TrimPrefix(s, "base64:")
				if dec, err := base64.StdEncoding.DecodeString(raw); err == nil {
					row[k] = dec
					continue
				}
			}
			if strings.HasPrefix(s, "hex:") {
				raw := strings.TrimPrefix(s, "hex:")
				if dec, err := hex.DecodeString(raw); err == nil {
					row[k] = dec
					continue
				}
			}
			row[k] = []byte(s)
		default:
			b, _ := json.Marshal(v)
			row[k] = b
		}
	}
}

// bcrypt
func hashPasswordFieldsIfPresent(row map[string]any, fields []string, cost int) error {
	for _, f := range fields {
		v, ok := row[f]
		if !ok || v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			continue
		}
		if looksBcrypt(s) {
			continue
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(s), cost)
		if err != nil {
			return err
		}
		row[f] = string(hashed)
	}
	return nil
}
func looksBcrypt(s string) bool {
	_, err := bcrypt.Cost([]byte(s))
	return err == nil
}
