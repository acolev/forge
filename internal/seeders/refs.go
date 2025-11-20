package seeders

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type refSpec struct {
	Table    string         `yaml:"table"`
	Where    map[string]any `yaml:"where"`
	Select   string         `yaml:"select"`
	Default  any            `yaml:"default,omitempty"`
	Required *bool          `yaml:"required,omitempty"`
}

func parseRefShortcut(s string) (refSpec, bool) {
	const pfx = "ref:"
	if !strings.HasPrefix(s, pfx) {
		return refSpec{}, false
	}
	body := strings.TrimPrefix(s, pfx)
	parts := strings.Split(body, "|")
	if len(parts) < 2 {
		return refSpec{}, false
	}
	spec := refSpec{Table: parts[0], Where: map[string]any{}}
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if i == len(parts)-1 && !strings.Contains(p, "=") {
			spec.Select = p
			break
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return refSpec{}, false
		}
		if strings.EqualFold(kv[0], "select") {
			spec.Select = kv[1]
			continue
		}
		spec.Where[kv[0]] = kv[1]
	}
	if spec.Select == "" {
		spec.Select = "id"
	}
	return spec, true
}

func fetchRefValue(tx *gorm.DB, spec refSpec) (any, bool, error) {
	if spec.Table == "" || spec.Select == "" || len(spec.Where) == 0 {
		return nil, false, fmt.Errorf("$ref: invalid spec (table/select/where required)")
	}
	q := tx.Table(spec.Table).Select(spec.Select)
	for k, v := range spec.Where {
		q = q.Where(fmt.Sprintf("%s = ?", k), v)
	}
	var vals []any
	if err := q.Limit(1).Pluck(spec.Select, &vals).Error; err != nil {
		return nil, false, err
	}
	if len(vals) == 0 {
		return nil, false, nil
	}
	return vals[0], true, nil
}

// Универсальный резолвер значения: объектный {"$ref": {...}} (включая вложенные where) и строковый "ref:..."
func resolveAnyRef(tx *gorm.DB, v any) (any, bool, error) {
	switch vv := v.(type) {
	case map[string]any:
		if raw, ok := vv["$ref"]; ok {
			inner, ok := raw.(map[string]any)
			if !ok {
				return nil, true, fmt.Errorf("$ref must be an object")
			}
			spec := refSpec{Where: map[string]any{}}
			if s, _ := inner["table"].(string); s != "" {
				spec.Table = s
			}
			if sel, _ := inner["select"].(string); sel != "" {
				spec.Select = sel
			} else {
				spec.Select = "id"
			}
			if req, ok := inner["required"].(bool); ok {
				spec.Required = &req
			}
			if def, ok := inner["default"]; ok {
				spec.Default = def
			}
			if w, ok := inner["where"].(map[string]any); ok {
				for wk, wv := range w {
					resolved, wasRef, err := resolveAnyRef(tx, wv)
					if err != nil {
						return nil, true, err
					}
					if wasRef {
						spec.Where[wk] = resolved
					} else {
						spec.Where[wk] = wv
					}
				}
			}
			val, found, err := fetchRefValue(tx, spec)
			if err != nil {
				return nil, true, err
			}
			required := true
			if spec.Required != nil {
				required = *spec.Required
			}
			if !found {
				if required && spec.Default == nil {
					return nil, true, fmt.Errorf("$ref not found (%s) where %v", spec.Table, spec.Where)
				}
				return spec.Default, true, nil
			}
			return val, true, nil
		}
	case string:
		if strings.HasPrefix(vv, "ref:") {
			spec, ok := parseRefShortcut(vv)
			if !ok {
				return nil, true, fmt.Errorf("invalid ref shortcut %q", vv)
			}
			val, found, err := fetchRefValue(tx, spec)
			if err != nil {
				return nil, true, err
			}
			if !found {
				return nil, true, fmt.Errorf("ref not found (%s) where %v", spec.Table, spec.Where)
			}
			return val, true, nil
		}
	}
	return v, false, nil
}

func resolveRowRefs(tx *gorm.DB, row map[string]any) error {
	for k, v := range row {
		if resolved, wasRef, err := resolveAnyRef(tx, v); wasRef {
			if err != nil {
				return fmt.Errorf("field %s: %w", k, err)
			}
			row[k] = resolved
		}
	}
	return nil
}
