package seeders

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CreateSeed(name, kind string) (string, error) {
	content, err := seedTemplate(kind, name)
	if err != nil {
		return "", err
	}
	return writeSeed(name, content)
}

// writeSeed writes seed YAML content to a timestamped file in the seeds dir.
func writeSeed(name, content string) (string, error) {
	if err := ensureSeedsDirectory(); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%d_%s.yaml", time.Now().Unix(), strings.TrimSpace(name))
	path := filepath.Join(seedsDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write seed file %s: %w", path, err)
	}
	return path, nil
}

func ensureSeedsDirectory() error {
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		return fmt.Errorf("create seeds directory: %w", err)
	}
	return nil
}

func seedTemplate(kind, name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "fixture":
		return fmt.Sprintf(`name: %s
type: fixture
table: %s
count: 10
template:
  name: "fake:full_name"
  email: "fake:email"
  phone: "fake:phone"
  is_active: "fake:bool"
`, name, inferSeedTable(name)), nil
	case "sql":
		return fmt.Sprintf(`name: %s
type: sql
sql: |
  -- write SQL here
`, name), nil
	case "go":
		return fmt.Sprintf(`name: %s
type: go
func: %s
`, name, inferGoFuncName(name)), nil
	default:
		return "", fmt.Errorf("unsupported seed type %q (expected: fixture, sql, go)", kind)
	}
}

func inferSeedTable(name string) string {
	clean := strings.TrimSpace(name)
	clean = strings.TrimSuffix(clean, ".yaml")
	clean = strings.TrimPrefix(clean, "seed_")
	clean = strings.TrimPrefix(clean, "create_")
	return strings.ReplaceAll(clean, "-", "_")
}

func inferGoFuncName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return "SeedData"
	}
	var b strings.Builder
	b.WriteString("Seed")
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}
