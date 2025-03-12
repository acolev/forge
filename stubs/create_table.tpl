-- UP
CREATE TABLE {table_name}(
   id INT,
   created_at TIMESTAMP,
   updated_at TIMESTAMP,
   deleted_at TIMESTAMP
);

-- DOWN
DROP TABLE IF EXISTS {table_name};