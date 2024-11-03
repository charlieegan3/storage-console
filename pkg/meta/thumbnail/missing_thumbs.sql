WITH has_thumbs AS (
    SELECT
        blobs.id,
        blobs.md5,
        objects.key,
        COALESCE(blob_metadata.thumbnail, false) AS has_thumb
    FROM objects
    JOIN object_blobs
        ON object_blobs.object_id = objects.id
    JOIN blobs
        ON object_blobs.blob_id = blobs.id
    LEFT JOIN blob_metadata
        ON blob_metadata.blob_id = blobs.id
    WHERE
      objects.deleted_at is NULL AND
      blobs.content_type_id IN (
        SELECT id
        FROM content_types
        WHERE name IN (
            'image/jpeg', 'image/jpg', 'image/jp2',
            'image/tiff',
            'image/png',
            'image/webp',
            'image/heic',
            'image/gif',
            'application/pdf'
        )
    )
)
SELECT
    id,
    md5,
    key
FROM has_thumbs
WHERE NOT has_thumb;
