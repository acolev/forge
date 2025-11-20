package migrations

import (
	"fmt"
	"os"
	"path/filepath"
)

func getTemplate(name string) (string, error) {
	path := filepath.Join("database/stubs", name+".stub.sql")
	if b, err := os.ReadFile(path); err == nil {
		return string(b), nil
	}

	if tpl, ok := builtinStubs[name]; ok {
		return tpl, nil
	}

	return "", fmt.Errorf("stub %q not found", name)
}

var builtinStubs = map[string]string{
	"create_table": `-- UP
CREATE TABLE {table_name} (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- add your columns here
);

-- DOWN
DROP TABLE IF EXISTS {table_name};
`,

	"update_table": `-- UP
-- ALTER TABLE {table_name} ADD COLUMN your_column datatype;
-- ALTER TABLE {table_name} ALTER COLUMN your_column TYPE new_datatype;

-- DOWN
-- ALTER TABLE {table_name} DROP COLUMN your_column;
`,

	"add_column": `-- UP
-- ALTER TABLE {table_name} ADD COLUMN your_column datatype;

-- DOWN
-- ALTER TABLE {table_name} DROP COLUMN your_column;
`,

	"drop_column": `-- UP
-- ALTER TABLE {table_name} DROP COLUMN your_column;

-- DOWN
-- ALTER TABLE {table_name} ADD COLUMN your_column datatype;
`,

	"create_pivot_table": `-- UP
CREATE TABLE {table_name} (
    left_id  BIGINT NOT NULL,
    right_id BIGINT NOT NULL,
    PRIMARY KEY (left_id, right_id)
    -- add FKs if needed
);

-- DOWN
DROP TABLE IF EXISTS {table_name};
`,

	"add_index": `-- UP
-- CREATE INDEX CONCURRENTLY idx_{table_name}_col ON {table_name} (col);

-- DOWN
-- DROP INDEX CONCURRENTLY IF EXISTS idx_{table_name}_col;
`,

	"drop_index": `-- UP
-- DROP INDEX CONCURRENTLY IF EXISTS idx_{table_name}_col;

-- DOWN
-- CREATE INDEX CONCURRENTLY idx_{table_name}_col ON {table_name} (col);
`,
}
