package importer

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/database"
)

type Options struct {
	StorageProviderName, BucketName, SchemaName string
}

func Run(ctx context.Context, db *sql.DB, minioClient *minio.Client, opts *Options) error {

	if opts.SchemaName == "" {
		return fmt.Errorf("schema name is required")
	}

	if opts.StorageProviderName == "" {
		return fmt.Errorf("storage provider name is required")
	}

	if opts.BucketName == "" {
		return fmt.Errorf("bucket name is required")
	}

	txn, err := database.NewTxnWithSchema(db, opts.SchemaName)
	if err != nil {
		return fmt.Errorf("could not start transaction: %s", err)
	}

	createObjectStorageProviderSQL := `
INSERT INTO object_storage_providers (name)
  VALUES ($1)
  ON CONFLICT (name) DO NOTHING;
`

	_, err = txn.Exec(createObjectStorageProviderSQL, opts.StorageProviderName)
	if err != nil {
		return fmt.Errorf("could not insert object storage provider: %s", err)
	}

	createBucketSQL := `
WITH object_storage_provider AS (
	SELECT id FROM object_storage_providers WHERE name = $1
	                                        LIMIT 1
)
INSERT INTO buckets (name, object_storage_provider_id)
  VALUES ($2, (SELECT id FROM object_storage_provider))
  ON CONFLICT (name) DO NOTHING;
`

	_, err = txn.Exec(createBucketSQL, opts.StorageProviderName, opts.BucketName)
	if err != nil {
		return fmt.Errorf("could not insert bucket: %s", err)
	}

	for obj := range minioClient.ListObjects(
		ctx,
		opts.BucketName,
		minio.ListObjectsOptions{Recursive: true},
	) {
		objData, err := minioClient.StatObject(ctx, opts.BucketName, obj.Key, minio.StatObjectOptions{})
		if err != nil {
			return fmt.Errorf("could not stat object: %s", err)
		}

		parts := strings.Split(obj.Key, "/")
		if len(parts) == 0 {
			continue
		}

		dirPath := filepath.Dir(obj.Key)
		if dirPath == "." {
			// then the item is in the root
			continue
		}

		objectInitSQL := `
with bucket as (
	SELECT id FROM buckets WHERE name = $1 LIMIT 1
),
dir as (
  select find_or_create_directory_in_bucket((select id from bucket), $2) as id
)
INSERT INTO objects (name, directory_id) VALUES ($3, (select id from dir))
ON CONFLICT (name, directory_id) DO NOTHING;
`
		_, err = txn.Exec(objectInitSQL, opts.BucketName, dirPath, filepath.Base(obj.Key))
		if err != nil {
			return fmt.Errorf("could not create object: %s", err)
		}

		blobInitSQL := `
INSERT INTO blobs
	(md5, size, last_modified, content_type_id)
VALUES ($1, $2, $3, find_or_create_content_type($4))
ON CONFLICT (md5) DO UPDATE
	SET size = EXCLUDED.size,
	    last_modified = EXCLUDED.last_modified,
	    content_type_id = EXCLUDED.content_type_id
RETURNING id;
`

		var blobID int
		err = txn.QueryRow(blobInitSQL, obj.ETag, obj.Size, obj.LastModified, objData.ContentType).Scan(&blobID)
		if err != nil {
			return fmt.Errorf("could not create blob: %s", err)
		}

		objectBlobSQL := `
with bucket as (
	SELECT id FROM buckets WHERE name = $1 LIMIT 1
),
dir as (
  select find_or_create_directory_in_bucket((select id from bucket), $2) as id
),
object as (
  select id from objects where name = $3 and directory_id = (select id from dir) limit 1
)
INSERT INTO object_blobs (object_id, blob_id) VALUES ((select id from object), $4)
ON CONFLICT (object_id, blob_id) DO NOTHING;
`
		_, err = txn.Exec(objectBlobSQL, opts.BucketName, dirPath, filepath.Base(obj.Key), blobID)
		if err != nil {
			return fmt.Errorf("could not create object blob: %s", err)
		}
	}

	err = txn.Commit()
	if err != nil {
		return fmt.Errorf("could not commit transaction: %s", err)
	}

	return nil
}
