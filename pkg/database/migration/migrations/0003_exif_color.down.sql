SET SCHEMA 'storage_console';
ALTER TABLE blob_metadata
DROP COLUMN exif,
DROP COLUMN color;
