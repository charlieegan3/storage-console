SET SCHEMA 'storage_console';
WITH needs_metadatas AS (
    SELECT
        blobs.id,
        blobs.md5,
        objects.key
    FROM objects
    JOIN object_blobs
        ON object_blobs.object_id = objects.id
    JOIN blobs
        ON object_blobs.blob_id = blobs.id
    LEFT JOIN blob_metadata
        ON blob_metadata.blob_id = blobs.id
    WHERE
      objects.deleted_at IS NULL AND
      (blob_metadata.%s = 'unknown' OR blob_metadata.%s is null) AND
      blobs.content_type_id IN (
        SELECT id
        FROM content_types
        WHERE name IN (%s)
    )
)
SELECT
    id,
    md5,
    key
FROM needs_metadatas

