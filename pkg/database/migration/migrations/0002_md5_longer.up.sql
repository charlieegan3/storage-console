SET SCHEMA 'storage_console';

ALTER TABLE blobs
-- md5 values are 32 characters long, but minio used a -N when files have been
-- multipart uploaded, which makes the ETag values longer.
ALTER COLUMN md5 TYPE VARCHAR(64);
