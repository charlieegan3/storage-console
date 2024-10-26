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
    WHERE blobs.content_type_id in (select id from content_types where name = 'image/jpeg')
)
SELECT
    id,
    md5,
    key
FROM has_thumbs
WHERE has_thumb IS false;
