CREATE SCHEMA IF NOT EXISTS storage_console;

SET SCHEMA 'storage_console';

CREATE TABLE IF NOT EXISTS tasks (
  id SERIAL PRIMARY KEY,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL,

  initiator VARCHAR(255) NOT NULL,
  status TEXT NOT NULL,
  operations INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS objects (
  id SERIAL PRIMARY KEY,
  key TEXT NOT NULL,
  deleted_at TIMESTAMP NULL,
  CHECK (key != ''),
  UNIQUE (key)
);

CREATE TABLE IF NOT EXISTS content_types (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL DEFAULT 'UNKNOWN',
  UNIQUE (name)
);

CREATE OR REPLACE FUNCTION find_or_create_content_type(
    content_type_name VARCHAR(255)
)
    RETURNS INTEGER AS $$
DECLARE
    ct_id INTEGER;
BEGIN
    SET SCHEMA 'storage_console';
    -- Check if the content type already exists
    SELECT id INTO ct_id
    FROM content_types
    WHERE name = content_type_name;

    IF ct_id IS NULL THEN
        -- If not, insert a new row and retrieve the ID
        INSERT INTO content_types (name)
        VALUES (content_type_name)
        RETURNING id INTO ct_id;
    END IF;

    RETURN ct_id;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS blobs (
   id SERIAL PRIMARY KEY,
   size BIGINT NOT NULL,
   last_modified TIMESTAMP NOT NULL,

   md5 VARCHAR(32) NOT NULL,
   UNIQUE (md5),

   content_type_id INTEGER NOT NULL,
   FOREIGN KEY (content_type_id) REFERENCES content_types(id)
);

CREATE TABLE IF NOT EXISTS object_blobs (
  object_id INTEGER NOT NULL,
  blob_id INTEGER NOT NULL,
  PRIMARY KEY (object_id, blob_id),
  FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE,
  FOREIGN KEY (blob_id) REFERENCES blobs(id)
);

CREATE TABLE IF NOT EXISTS blob_metadata (
  id SERIAL PRIMARY KEY,
  blob_id INTEGER NOT NULL,
  thumbnail BOOLEAN NOT NULL DEFAULT FALSE,
  FOREIGN KEY (blob_id) REFERENCES blobs(id) ON DELETE CASCADE,
  UNIQUE (blob_id)
);
