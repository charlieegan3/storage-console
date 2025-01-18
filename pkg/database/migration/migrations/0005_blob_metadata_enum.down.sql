SET SCHEMA 'storage_console';

BEGIN;

-- Add back the old boolean columns with the correct type and default value
ALTER TABLE blob_metadata
ADD COLUMN thumbnail_old BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE blob_metadata
ADD COLUMN exif_old BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE blob_metadata
ADD COLUMN color_old BOOLEAN NOT NULL DEFAULT FALSE;

-- Migrate the data back to the boolean columns
UPDATE blob_metadata
SET
    thumbnail_old = CASE
        WHEN thumbnail = 'success' THEN TRUE
        ELSE FALSE
    END,
    exif_old = CASE
        WHEN exif = 'success' THEN TRUE
        ELSE FALSE
    END,
    color_old = CASE
        WHEN color = 'success' THEN TRUE
        ELSE FALSE
    END;

-- Drop the ENUM-typed columns
ALTER TABLE blob_metadata
DROP COLUMN thumbnail;

ALTER TABLE blob_metadata
DROP COLUMN exif;

ALTER TABLE blob_metadata
DROP COLUMN color;

-- Rename the old boolean columns back to their original names
ALTER TABLE blob_metadata
RENAME COLUMN thumbnail_old TO thumbnail;

ALTER TABLE blob_metadata
RENAME COLUMN exif_old TO exif;

ALTER TABLE blob_metadata
RENAME COLUMN color_old TO color;

-- Optionally drop the ENUM type if no longer used
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'blob_metadata_result'
    ) THEN
        DROP TYPE blob_metadata_result;
    END IF;
END $$;

COMMIT;
