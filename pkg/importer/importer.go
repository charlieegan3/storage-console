package importer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

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

	ObjectsDeleted int
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

	exists, err := minioClient.BucketExists(ctx, opts.BucketName)
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket exists: %s", err)
	}

	if !exists {
		return nil, fmt.Errorf("bucket does not exist")
	}

	taskInsertSQL := `
SET SCHEMA 'storage_console';
INSERT INTO tasks (initiator, status) VALUES ('importer', 'starting')
RETURNING id;
`
	var taskID int
	err = db.QueryRow(taskInsertSQL).Scan(&taskID)
	if err != nil {
		return nil, fmt.Errorf("could not insert task: %s", err)
	}

	txn, err := database.NewTxnWithSchema(db, opts.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %s", err)
	}

	err = updateTask(db, taskID, "transaction created", false, false)
	if err != nil {
		return nil, fmt.Errorf("could not update task: %s", err)
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

	if r.ProviderCreated {
		err = updateTask(db, taskID, "provider created", true, false)
		if err != nil {
			return nil, fmt.Errorf("could not update task: %s", err)
		}
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

	if r.BucketCreated {
		err = updateTask(db, taskID, "bucket created", true, false)
		if err != nil {
			return nil, fmt.Errorf("could not update task: %s", err)
		}
	}

	bucketIDSQL := `
WITH object_storage_provider AS (
	SELECT id FROM object_storage_providers WHERE name = $1
	                                        LIMIT 1
),
bucket AS (
	SELECT id FROM buckets WHERE name = $2
	    AND object_storage_provider_id = (SELECT id FROM object_storage_provider)	
		LIMIT 1
)
SELECT id FROM bucket;
`
	var bucketID int
	err = txn.QueryRow(bucketIDSQL, opts.StorageProviderName, opts.BucketName).Scan(&bucketID)
	if err != nil {
		return nil, fmt.Errorf("could not select bucket ID: %s", err)
	}

	existingPathSQL := `
WITH RECURSIVE directory_path AS (
  SELECT 
    id,
    name,
    directory_id as parent_directory_id,
    CAST(name AS VARCHAR) AS path
  FROM 
    objects

  UNION ALL

  SELECT 
    d.id,
    d.name,
    d.parent_directory_id,
    d.name || '/' || p.path AS path
  FROM 
    directories d
  JOIN 
    directory_path p ON d.id = p.parent_directory_id
)
SELECT 
  path
FROM 
  directory_path
WHERE 
  parent_directory_id = (select id from directories where bucket_id = $1 and parent_directory_id IS NULL);
`

	rows, err := txn.Query(existingPathSQL, bucketID)
	if err != nil {
		return nil, fmt.Errorf("could not select existing paths: %s", err)
	}

	pathsToRemove := make(map[string]bool)
	for rows.Next() {
		var path string
		err = rows.Scan(&path)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not scan path: %s", err)
		}
		pathsToRemove[path] = true
	}

	err = updateTask(db, taskID, "existing state scanned", false, false)
	if err != nil {
		return nil, fmt.Errorf("could not update task: %s", err)
	}

	for obj := range minioClient.ListObjects(
		ctx,
		opts.BucketName,
		minio.ListObjectsOptions{Recursive: true},
	) {
		if _, ok := pathsToRemove[obj.Key]; ok {
			pathsToRemove[obj.Key] = false
		}
		dirPath := filepath.Dir(obj.Key)

		objectInitSQL := `
with dir as (
  select find_or_create_directory_in_bucket($1, $2) as id
)
INSERT INTO objects (name, directory_id) VALUES ($3, (select id from dir))
ON CONFLICT (name, directory_id) DO NOTHING;
`
		result, err = txn.Exec(objectInitSQL, bucketID, dirPath, filepath.Base(obj.Key))
		if err != nil {
			return nil, fmt.Errorf("could not create object: %s", err)
		}
		objCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if object was created: %s", err)
		}
		if objCreated {
			err = updateTask(db, taskID, fmt.Sprintf("object created: %s", obj.Key), true, false)
			if err != nil {
				return nil, fmt.Errorf("could not update task: %s", err)
			}

			r.ObjectsCreated++
		}

		findExistingBlobSQL := `
with dir as (
  select find_or_create_directory_in_bucket($1, $2) as id
),
object as (
  select id from objects where name = $3 and directory_id = (select id from dir) limit 1
),
blob as (
  select blob_id from object_blobs where object_id = (select id from object) limit 1
)
select md5, size, last_modified, content_type_id from blobs where id = (select blob_id from blob) limit 1;
`
		var md5 string
		var size int64
		var lastModified time.Time
		var contentTypeID int
		err = txn.QueryRow(findExistingBlobSQL, bucketID, dirPath, filepath.Base(obj.Key)).Scan(&md5, &size, &lastModified, &contentTypeID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed checking presence of blob: %s", err)
		}

		selectContentTypeSQL := `
SELECT name FROM content_types WHERE id = $1;
`
		var contentType string
		err = txn.QueryRow(selectContentTypeSQL, contentTypeID).Scan(&contentType)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("could not select content type: %s", err)
		}

		if md5 != obj.ETag {
			r.ObjectStatCalls++

			objData, err := minioClient.StatObject(ctx, opts.BucketName, obj.Key, minio.StatObjectOptions{})
			if err != nil {
				return nil, fmt.Errorf("could not stat object: %s", err)
			}
			if objData.ETag != obj.ETag {
				return nil, fmt.Errorf("unexpected ETag: %s != %s", objData.ETag, obj.ETag)
			}

			md5 = obj.ETag
			size = obj.Size
			lastModified = obj.LastModified
			contentType = objData.ContentType
		}

		blobInitSQL := `
INSERT INTO blobs
	(md5, size, last_modified, content_type_id)
VALUES ($1, $2, $3, find_or_create_content_type($4))
ON CONFLICT (md5) DO NOTHING;
`

		result, err = txn.Exec(blobInitSQL, md5, size, lastModified, contentType)
		if err != nil {
			return nil, fmt.Errorf("could not create blob: %s", err)
		}
		blobCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if blob was created: %s", err)
		}
		if blobCreated {
			r.BlobsCreated++

			err = updateTask(db, taskID, fmt.Sprintf("blob created: %s", md5), true, false)
			if err != nil {
				return nil, fmt.Errorf("could not update task: %s", err)
			}
		}

		var blobID int
		err = txn.QueryRow("SELECT id FROM blobs WHERE md5 = $1", md5).Scan(&blobID)
		if err != nil {
			return nil, fmt.Errorf("could not select blob ID: %s", err)
		}

		objectBlobSQL := `
with dir as (
  select find_or_create_directory_in_bucket($1, $2) as id
),
object as (
  select id from objects where name = $3 and directory_id = (select id from dir) limit 1
)
INSERT INTO object_blobs (object_id, blob_id) VALUES ((select id from object), $4)
ON CONFLICT (object_id, blob_id) DO NOTHING;
`
		result, err = txn.Exec(objectBlobSQL, bucketID, dirPath, filepath.Base(obj.Key), blobID)
		if err != nil {
			return nil, fmt.Errorf("could not create object blob: %s", err)
		}
		objBlobCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if object blob was created: %s", err)
		}
		if objBlobCreated {
			r.BlobsLinked++

			err = updateTask(db, taskID, fmt.Sprintf("object blob linked: %s", obj.Key), true, false)
			if err != nil {
				return nil, fmt.Errorf("could not update task: %s", err)
			}
		}
	}

	for path, toRemove := range pathsToRemove {
		if !toRemove {
			continue
		}

		deleteObjectSQL := `
with dir as (
    select find_or_create_directory_in_bucket($1, $2) as id
)
delete from object_blobs where object_id in (select id from objects where name = $3 and directory_id = (select id from dir));
`
		_, err = txn.Exec(deleteObjectSQL, bucketID, filepath.Dir(path), filepath.Base(path))
		if err != nil {
			return nil, fmt.Errorf("could not delete object: %s", err)
		}

		r.ObjectsDeleted++

		err = updateTask(db, taskID, fmt.Sprintf("object deleted: %s", path), true, false)
		if err != nil {
			return nil, fmt.Errorf("could not update task: %s", err)
		}
	}

	// select all objects that do not have an object blob
	deleteDisattachedObjectsSQL := `
delete from objects where id not in (
  select object_id from object_blobs
)
`
	_, err = txn.Exec(deleteDisattachedObjectsSQL)
	if err != nil {
		return nil, fmt.Errorf("could not delete disattached objects: %s", err)
	}

	err = txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %s", err)
	}

	err = updateTask(db, taskID, "completed", false, true)
	if err != nil {
		return nil, fmt.Errorf("could not update task: %s", err)
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

func updateTask(db *sql.DB, taskID int, status string, incOperations bool, complete bool) error {

	var increment int
	if incOperations {
		increment = 1
	}
	var completedAt string
	if complete {
		completedAt = ", completed_at = CURRENT_TIMESTAMP"
	}

	updateTaskSQL := fmt.Sprintf(`
UPDATE storage_console.tasks SET status = $1, operations = operations + $3 %s WHERE id = $2;
`, completedAt)
	_, err := db.Exec(updateTaskSQL, status, taskID, increment)
	if err != nil {
		return fmt.Errorf("could not update task: %s", err)
	}

	return nil
}
