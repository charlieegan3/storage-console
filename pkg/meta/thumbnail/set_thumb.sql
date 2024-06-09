INSERT INTO blob_metadata (blob_id, thumbnail)
VALUES ($1, true)
ON CONFLICT (blob_id) 
DO UPDATE SET thumbnail = true;