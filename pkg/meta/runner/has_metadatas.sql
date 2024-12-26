WITH has_metadatas AS (
    SELECT
        blobs.id,
        blobs.md5,
        objects.key,
        COALESCE(blob_metadata.%s, false) AS has_metadata
    FROM objects
    JOIN object_blobs
        ON object_blobs.object_id = objects.id
    JOIN blobs
        ON object_blobs.blob_id = blobs.id
    LEFT JOIN blob_metadata
        ON blob_metadata.blob_id = blobs.id
    WHERE
      objects.deleted_at IS NULL AND
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
FROM has_metadatas
WHERE NOT has_metadata;
