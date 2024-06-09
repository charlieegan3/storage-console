WITH bucket AS (
    SELECT buckets.id 
    FROM buckets
    JOIN object_storage_providers 
        ON object_storage_providers.id = buckets.object_storage_provider_id
    WHERE buckets.name = $1
      AND object_storage_providers.name = $2
    LIMIT 1
),
has_thumbs AS (
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
    WHERE bucket_id IN (SELECT id FROM bucket)
    and blobs.content_type_id in (select id from content_types where name = 'image/jpeg')
)
SELECT 
    id,
    md5, 
    key 
FROM has_thumbs
WHERE has_thumb IS false;