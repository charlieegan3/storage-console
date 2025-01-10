SELECT
    bm.blob_id,
    blobs.md5,
    EXISTS (
        SELECT 1
        FROM blob_properties bp
        WHERE bp.blob_id = bm.blob_id
          AND bp.source = 'exif'
          AND bp.property_type = 'Done'
    ) AS exif_set,
    EXISTS (
        SELECT 1
        FROM blob_properties bp
        WHERE bp.blob_id = bm.blob_id
          AND bp.source = 'color'
          AND bp.property_type = 'Done'
    ) AS color_set
FROM
    blob_metadata bm
JOIN
    blobs on bm.blob_id = blobs.id
WHERE
    bm.exif = TRUE OR bm.color = TRUE
ORDER BY
    bm.blob_id;
