-- UP
CREATE TABLE {table_name}
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- DOWN
DROP TABLE IF EXISTS {table_name};