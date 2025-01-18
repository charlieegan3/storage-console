SELECT
    bm.blob_id,
    objects.key,
    blobs.md5,
    (bm.exif = 'success' AND NOT EXISTS (
        SELECT 1
        FROM blob_properties bp
        WHERE bp.blob_id = bm.blob_id
          AND bp.source = 'exif'
          AND bp.property_type = 'Done'
          AND bm.exif = 'success'
    )) AS exif_missing,
    (bm.color = 'success' AND NOT EXISTS (
        SELECT 1
        FROM blob_properties bp
        WHERE bp.blob_id = bm.blob_id
          AND bp.source = 'color'
          AND bp.property_type = 'Done'
          AND bm.color = 'success'
    )) AS color_missing
FROM
    blob_metadata bm
JOIN
    blobs ON bm.blob_id = blobs.id
JOIN
    object_blobs ON blobs.id = object_blobs.blob_id
JOIN
    objects ON object_blobs.object_id = objects.id
WHERE
    bm.exif = 'success' OR bm.color = 'success'
ORDER BY
    bm.blob_id;
