SET SCHEMA 'storage_console';

ALTER TABLE blobs
ALTER COLUMN md5 TYPE VARCHAR(32);
