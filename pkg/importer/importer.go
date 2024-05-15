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

type Report struct {
	ObjectStatCalls int

	ProviderCreated bool
	BucketCreated   bool

	ObjectsCreated int
	BlobsCreated   int
	BlobsLinked    int
}

func Run(ctx context.Context, db *sql.DB, minioClient *minio.Client, opts *Options) (*Report, error) {

	if opts.SchemaName == "" {
		return nil, fmt.Errorf("schema name is required")
	}

	if opts.StorageProviderName == "" {
		return nil, fmt.Errorf("storage provider name is required")
	}

	if opts.BucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	txn, err := database.NewTxnWithSchema(db, opts.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %s", err)
	}

	shouldRollback := true
	defer func() {
		if shouldRollback {
			err := txn.Rollback()
			if err != nil {
				fmt.Printf("could not rollback transaction: %s", err)
			}
		}
	}()

	var r Report

	createObjectStorageProviderSQL := `
INSERT INTO object_storage_providers (name)
  VALUES ($1)
  ON CONFLICT (name) DO NOTHING;
`

	result, err := txn.Exec(createObjectStorageProviderSQL, opts.StorageProviderName)
	if err != nil {
		return nil, fmt.Errorf("could not insert object storage provider: %s", err)
	}
	r.ProviderCreated, err = didUpdate(result)
	if err != nil {
		return nil, fmt.Errorf("could not check if provider was created: %s", err)
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

	result, err = txn.Exec(createBucketSQL, opts.StorageProviderName, opts.BucketName)
	if err != nil {
		return nil, fmt.Errorf("could not insert bucket: %s", err)
	}
	r.BucketCreated, err = didUpdate(result)
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket was created: %s", err)
	}

	exists, err := minioClient.BucketExists(ctx, opts.BucketName)
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket exists: %s", err)
	}

	if !exists {
		return nil, fmt.Errorf("bucket does not exist")
	}

	for obj := range minioClient.ListObjects(
		ctx,
		opts.BucketName,
		minio.ListObjectsOptions{Recursive: true},
	) {
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
		result, err = txn.Exec(objectInitSQL, opts.BucketName, dirPath, filepath.Base(obj.Key))
		if err != nil {
			return nil, fmt.Errorf("could not create object: %s", err)
		}
		objCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if object was created: %s", err)
		}
		if objCreated {
			r.ObjectsCreated++
		}

		findExistingBlobSQL := `
with bucket as (
	SELECT id FROM buckets WHERE name = $1 LIMIT 1
),
dir as (
  select find_or_create_directory_in_bucket((select id from bucket), $2) as id
),
object as (
  select id from objects where name = $3 and directory_id = (select id from dir) limit 1
),
blob as (
  select blob_id from object_blobs where object_id = (select id from object) limit 1
)
select md5 from blobs where id = (select blob_id from blob);
`
		var md5 string
		err = txn.QueryRow(findExistingBlobSQL, opts.BucketName, dirPath, filepath.Base(obj.Key)).Scan(&md5)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed checking presence of blob: %s", err)
		}

		if md5 != "" && md5 == obj.ETag {
			// then the blob already exists
			continue
		}

		r.ObjectStatCalls++

		objData, err := minioClient.StatObject(ctx, opts.BucketName, obj.Key, minio.StatObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("could not stat object: %s", err)
		}
		if objData.ETag != obj.ETag {
			return nil, fmt.Errorf("unexpected ETag: %s != %s", objData.ETag, obj.ETag)
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
			return nil, fmt.Errorf("could not create blob: %s", err)
		}
		r.BlobsCreated++

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
		result, err = txn.Exec(objectBlobSQL, opts.BucketName, dirPath, filepath.Base(obj.Key), blobID)
		if err != nil {
			return nil, fmt.Errorf("could not create object blob: %s", err)
		}
		objBlobCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if object blob was created: %s", err)
		}
		if objBlobCreated {
			r.BlobsLinked++
		}
	}

	err = txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %s", err)
	}

	shouldRollback = false

	return &r, nil
}

func didUpdate(result sql.Result) (bool, error) {
	if result == nil {
		return false, fmt.Errorf("db result was missing")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get rows affected: %s", err)
	}

	return rowsAffected > 0, nil
}
