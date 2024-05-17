package importer

import (
	"bytes"
	"context"
	"testing"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/test"
)

func TestBucketKeys(t *testing.T) {
	var ctx = context.Background()

	minioClient, minioCleanup, err := test.InitMinio(ctx, t)
	defer func() {
		if minioCleanup == nil {
			return
		}
		if err := minioCleanup(); err != nil {
			t.Fatalf("Could not cleanup minio: %s", err)
		}
	}()
	if err != nil {
		t.Fatalf("Could not init minio: %s", err)
	}

	db, postgresCleanup, err := test.InitPostgres(ctx, t)
	defer func() {
		if postgresCleanup == nil {
			return
		}
		if err := postgresCleanup(); err != nil {
			t.Fatalf("Could not cleanup postgres: %s", err)
		}
	}()
	if err != nil {
		t.Fatalf("Could not init database: %s", err)
	}

	// create the initial bucket state
	err = minioClient.MakeBucket(ctx, "example", minio.MakeBucketOptions{})
	if err != nil {
		t.Fatalf("Could not create bucket: %s", err)
	}

	var contentString = "hello"
	for _, v := range []string{"foo/bar.jpg", "bar/foo.jpg", "foo/bar/baz.jpg", "foo.jpg"} {
		_, err = minioClient.PutObject(
			ctx,
			"example",
			v,
			bytes.NewReader([]byte(contentString)),
			int64(len(contentString)),
			minio.PutObjectOptions{
				ContentType: "image/jpeg",
			},
		)
		if err != nil {
			t.Fatalf("Could not put object: %s", err)
		}
	}

	// run the importer
	report, err := Run(ctx, db, minioClient, &Options{
		BucketName:          "example",
		SchemaName:          "storage_console",
		StorageProviderName: "local-minio",
	})
	if err != nil {
		t.Fatalf("Could not run import: %s", err)
	}
	if report.ProviderCreated == false {
		t.Fatalf("Expected provider to be created")
	}
	if report.BucketCreated == false {
		t.Fatalf("Expected bucket to be created")
	}
	if report.ObjectsCreated != 3 {
		t.Fatalf("Expected 3 objects to be created, got %d", report.ObjectsCreated)
	}
	if report.BlobsCreated != 3 {
		t.Fatalf("Expected 3 blobs to be created, got %d", report.BlobsCreated)
	}
	if report.BlobsLinked != 3 {
		t.Fatalf("Expected 3 blobs to be linked, got %d", report.BlobsLinked)
	}
	if report.ObjectStatCalls != 3 {
		t.Fatalf("Expected 3 object stat calls, got %d", report.ObjectStatCalls)
	}

	// run again to test for idempotency
	report, err = Run(ctx, db, minioClient, &Options{
		BucketName:          "example",
		SchemaName:          "storage_console",
		StorageProviderName: "local-minio",
	})
	if err != nil {
		t.Fatalf("Could not run import: %s", err)
	}
	if report.ProviderCreated == true {
		t.Fatalf("Expected provider to not be created")
	}
	if report.BucketCreated == true {
		t.Fatalf("Expected bucket to not be created")
	}
	if report.ObjectsCreated != 0 {
		t.Fatalf("Expected 0 objects to be created, got %d", report.ObjectsCreated)
	}
	if report.BlobsCreated != 0 {
		t.Fatalf("Expected 0 blobs to be created, got %d", report.BlobsCreated)
	}
	if report.BlobsLinked != 0 {
		t.Fatalf("Expected 0 blobs to be linked, got %d", report.BlobsLinked)
	}
	if report.ObjectStatCalls != 0 {
		t.Fatalf("Expected 0 object stat calls, got %d", report.ObjectStatCalls)
	}

	// test the state in the database
	testSQL := `set schema 'storage_console';
select
  (select count(id) from blobs) as blob_count,
  (select count(id) from buckets) as bucket_count,
  (select count(id) from content_types) as content_type_count,
  (select count(id) from directories) as directory_count,
  (select count(*) from object_blobs) as object_blob_count,
  (select count(id) from object_storage_providers) as object_storage_provider_count,
  (select count(id) from objects) as object_count;
`

	var blobCount, bucketCount, contentTypeCount, directoryCount, objectBlobCount, objectStorageProviderCount, objectCount int
	err = db.QueryRow(testSQL).Scan(&blobCount, &bucketCount, &contentTypeCount, &directoryCount, &objectBlobCount, &objectStorageProviderCount, &objectCount)
	if err != nil {
		t.Fatalf("Could not run test SQL: %s", err)
	}

	if blobCount != 1 {
		t.Fatalf("Expected 1 blob, got %d", blobCount)
	}

	if bucketCount != 1 {
		t.Fatalf("Expected 1 bucket, got %d", bucketCount)
	}

	if contentTypeCount != 1 {
		t.Fatalf("Expected 1 content type, got %d", contentTypeCount)
	}

	if directoryCount != 4 {
		t.Fatalf("Expected 4 directories, got %d", directoryCount)
	}

	if objectBlobCount != 3 {
		t.Fatalf("Expected 3 object blobs, got %d", objectBlobCount)
	}

	if objectStorageProviderCount != 1 {
		t.Fatalf("Expected 1 object storage provider, got %d", objectStorageProviderCount)
	}

	if objectCount != 3 {
		t.Fatalf("Expected 3 objects, got %d", objectCount)
	}

	testContentsSQL := `
with
  blob as (select * from blobs where md5 = $1),
  objBlob as (select * from object_blobs where blob_id = (select id from blob) and object_id = $2),
  obj as (select * from objects where id = (select object_id from objBlob)),
  dir as (select * from directories where id = (select directory_id from obj)),
  bucket as (select * from buckets where id = (select bucket_id from dir))
select (select name from bucket) as bucket, (select name from dir) as dir, (select name from obj) as obj;`

	var bucket, dir, obj string
	err = db.QueryRow(testContentsSQL, "5d41402abc4b2a76b9719d911017c592", 3).Scan(&bucket, &dir, &obj)
	if err != nil {
		t.Fatalf("Could not run test contents SQL: %s", err)
	}

	if bucket != "example" {
		t.Fatalf("Expected bucket to be example, got %s", bucket)
	}

	if dir != "bar" {
		t.Fatalf("Expected dir to be foo/bar, got %s", dir)
	}

	if obj != "baz.jpg" {
		t.Fatalf("Expected obj to be baz.jpg, got %s", obj)
	}

	testPathSQL := `
WITH RECURSIVE directory_path AS (
  SELECT 
    id,
    name,
    directory_id as parent_directory_id,
    CAST(name AS VARCHAR) AS path
  FROM 
    objects
  WHERE 
    id = $1

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
  parent_directory_id = (select id from directories where bucket_id = 1 and parent_directory_id IS NULL);
`

	var path string
	err = db.QueryRow(testPathSQL, 3).Scan(&path)
	if err != nil {
		t.Fatalf("Could not run test path SQL: %s", err)
	}

	if path != "foo/bar/baz.jpg" {
		t.Fatalf("Expected path to be bar, got %s", path)
	}

	// update an object
	var newContentString = "hello2"
	_, err = minioClient.PutObject(
		ctx,
		"example",
		"foo/bar/baz.jpg",
		bytes.NewReader([]byte(newContentString)),
		int64(len(newContentString)),
		minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
	)
	if err != nil {
		t.Fatalf("Could not put object: %s", err)
	}

	// run the importer
	report, err = Run(ctx, db, minioClient, &Options{
		BucketName:          "example",
		SchemaName:          "storage_console",
		StorageProviderName: "local-minio",
	})
	if err != nil {
		t.Fatalf("Could not run import: %s", err)
	}
	if report.BlobsCreated != 1 {
		t.Fatalf("Expected 1 blob to be created, got %d", report.BlobsCreated)
	}
	if report.ObjectStatCalls != 1 {
		t.Fatalf("Expected 1 object stat calls, got %d", report.ObjectStatCalls)
	}
}
