CREATE SCHEMA IF NOT EXISTS storage_console;

SET SCHEMA 'storage_console';

CREATE TABLE IF NOT EXISTS object_storage_providers (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS buckets (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,

  object_storage_provider_id INTEGER NOT NULL,
  FOREIGN KEY (object_storage_provider_id) REFERENCES object_storage_providers(id),

  UNIQUE (name),
  CHECK (name != '')
);

CREATE TABLE IF NOT EXISTS directories (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,

  bucket_id INTEGER NOT NULL,
  FOREIGN KEY (bucket_id) REFERENCES buckets(id),

  parent_directory_id INTEGER NULL,
  FOREIGN KEY (parent_directory_id) REFERENCES directories(id),

  UNIQUE (name, bucket_id, parent_directory_id),
  CHECK (name = 'root' OR parent_directory_id IS NOT NULL),
  CHECK (name != '')
);
CREATE UNIQUE INDEX root_dir_per_bucket ON directories(bucket_id) WHERE name = 'root';

CREATE OR REPLACE FUNCTION find_or_create_directory_in_bucket(bid INT, path TEXT) RETURNS INT AS $$
DECLARE
    path_elements TEXT[];
    current_parent_id INT;
    current_directory_id INT;
    element TEXT;
    last_directory_id INT;
BEGIN
    SET SCHEMA 'storage_console';
    -- Split the path into elements
    path_elements := string_to_array(path, '/');

    -- Find the 'root' directory id to start with
    SELECT id INTO current_parent_id FROM directories WHERE name = 'root' AND bucket_id = bid;

    IF current_parent_id IS NULL THEN
        INSERT INTO directories (name, bucket_id)
        VALUES ('root', bid)
        RETURNING id into current_parent_id;
    END IF;

    -- Loop through each element in the path array
    FOREACH element IN ARRAY path_elements LOOP
            IF element = '' THEN
                CONTINUE;
            END IF;

            -- Check if the directory already exists
            SELECT id INTO current_directory_id FROM directories
            WHERE name = element AND parent_directory_id = current_parent_id AND bucket_id = bid;

            -- If the directory does not exist, create it
            IF current_directory_id IS NULL THEN
                INSERT INTO directories (name, parent_directory_id, bucket_id)
                VALUES (element, current_parent_id, bid)
                RETURNING id INTO current_directory_id;
            END IF;

            -- Update the parent id for the next iteration
            current_parent_id := current_directory_id;
            -- Keep track of the last directory created
            last_directory_id := current_directory_id;
        END LOOP;

    -- Return the ID of the last directory created
    RETURN last_directory_id;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS objects (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,

  directory_id INTEGER NOT NULL,
  FOREIGN KEY (directory_id) REFERENCES directories(id),

  CHECK (name != ''),
  UNIQUE (name, directory_id)
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
  FOREIGN KEY (object_id) REFERENCES objects(id),
  FOREIGN KEY (blob_id) REFERENCES blobs(id)
);
