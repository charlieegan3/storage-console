package importer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/database"
)

const dataPath = "data/"

type Options struct {
	BucketName string
	SchemaName string
}

type Report struct {
	ObjectStatCalls int

	ObjectsCreated int
	BlobsCreated   int
	BlobsLinked    int

	ObjectsDeleted int
}

func Run(ctx context.Context, db *sql.DB, minioClient *minio.Client, opts *Options) (*Report, error) {
	if opts.SchemaName == "" {
		return nil, fmt.Errorf("schema name is required")
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

	existingPathSQL := `
select key from objects;
`

	rows, err := txn.Query(existingPathSQL)
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
		minio.ListObjectsOptions{Prefix: dataPath, Recursive: true},
	) {

		key := strings.TrimPrefix(obj.Key, dataPath)

		if _, ok := pathsToRemove[key]; ok {
			pathsToRemove[key] = false
		}

		objectInitSQL := `
INSERT INTO objects (key) VALUES ($1)
ON CONFLICT (key) DO NOTHING;
`
		result, err := txn.Exec(objectInitSQL, key)
		if err != nil {
			return nil, fmt.Errorf("could not create object: %s", err)
		}
		objCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if object was created: %s", err)
		}
		if objCreated {
			err = updateTask(db, taskID, fmt.Sprintf("object created: %s", key), true, false)
			if err != nil {
				return nil, fmt.Errorf("could not update task: %s", err)
			}

			r.ObjectsCreated++
		}

		findExistingObjectSQL := `
select id from objects where key = $1;
`
		var objectID int
		err = txn.QueryRow(findExistingObjectSQL, key).Scan(&objectID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get object: %s", err)
		}

		findExistingBlobSQL := `
select id from blobs where md5 = $1;
`
		var blobID int
		err = txn.QueryRow(findExistingBlobSQL, obj.ETag).Scan(&blobID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed checking presence of blob: %s", err)
		}
		if errors.Is(err, sql.ErrNoRows) {
			r.ObjectStatCalls++

			objData, err := minioClient.StatObject(
				ctx,
				opts.BucketName,
				path.Join(dataPath, key),
				minio.StatObjectOptions{},
			)
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
RETURNING id;
`

			err = txn.QueryRow(blobInitSQL, obj.ETag, obj.Size, obj.LastModified, objData.ContentType).Scan(&blobID)
			if err != nil {
				return nil, fmt.Errorf("could not create blob: %s", err)
			}

			r.BlobsCreated++

			err = updateTask(db, taskID, fmt.Sprintf("blob created: %s", obj.ETag), true, false)
			if err != nil {
				return nil, fmt.Errorf("could not update task: %s", err)
			}
		}

		err = txn.QueryRow("SELECT id FROM blobs WHERE md5 = $1", obj.ETag).Scan(&blobID)
		if err != nil {
			return nil, fmt.Errorf("could not select blob ID: %s", err)
		}

		objectBlobSQL := `
INSERT INTO object_blobs (object_id, blob_id) VALUES ($1, $2)
ON CONFLICT (object_id, blob_id) DO NOTHING;
`
		result, err = txn.Exec(objectBlobSQL, objectID, blobID)
		if err != nil {
			return nil, fmt.Errorf("could not create object blob: %s", err)
		}
		objBlobCreated, err := didUpdate(result)
		if err != nil {
			return nil, fmt.Errorf("could not check if object blob was created: %s", err)
		}
		if objBlobCreated {
			r.BlobsLinked++

			err = updateTask(db, taskID, fmt.Sprintf("object blob linked: %s", key), true, false)
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
update objects SET deleted_at = CURRENT_TIMESTAMP
where key = $1
`
		_, err = txn.Exec(deleteObjectSQL, path)
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
