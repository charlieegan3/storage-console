SET SCHEMA 'storage_console';
ALTER TABLE blob_metadata
ADD COLUMN exif BOOLEAN DEFAULT FALSE NOT NULL,
ADD COLUMN color BOOLEAN DEFAULT FALSE NOT NULL;
