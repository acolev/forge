-- UP
ALTER TABLE {table_name} ADD COLUMN new_column_name column_type;

-- DOWN
ALTER TABLE {table_name} DROP COLUMN new_column_name;