SET SCHEMA 'storage_console';

BEGIN;

-- Create the new enum type if not already created
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'blob_metadata_result'
    ) THEN
        CREATE TYPE blob_metadata_result AS ENUM ('success', 'failure', 'unknown');
    END IF;
END $$;

-- Add new columns with the new ENUM type (one column at a time)
ALTER TABLE blob_metadata
ADD COLUMN thumbnail_new blob_metadata_result DEFAULT 'unknown';

ALTER TABLE blob_metadata
ADD COLUMN exif_new blob_metadata_result DEFAULT 'unknown';

ALTER TABLE blob_metadata
ADD COLUMN color_new blob_metadata_result DEFAULT 'unknown';

-- Migrate the existing data with explicit casting to the ENUM type
UPDATE blob_metadata
SET
    thumbnail_new = CASE
        WHEN thumbnail THEN 'success'::blob_metadata_result
        ELSE 'unknown'::blob_metadata_result
    END,
    exif_new = CASE
        WHEN exif THEN 'success'::blob_metadata_result
        ELSE 'unknown'::blob_metadata_result
    END,
    color_new = CASE
        WHEN color THEN 'success'::blob_metadata_result
        ELSE 'unknown'::blob_metadata_result
    END;

-- Drop the old columns (one column at a time)
ALTER TABLE blob_metadata
DROP COLUMN thumbnail;

ALTER TABLE blob_metadata
DROP COLUMN exif;

ALTER TABLE blob_metadata
DROP COLUMN color;

-- Rename new columns to replace the old ones (one column at a time)
ALTER TABLE blob_metadata
RENAME COLUMN thumbnail_new TO thumbnail;

ALTER TABLE blob_metadata
RENAME COLUMN exif_new TO exif;

ALTER TABLE blob_metadata
RENAME COLUMN color_new TO color;

COMMIT;
