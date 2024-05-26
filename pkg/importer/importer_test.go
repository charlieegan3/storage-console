package importer

import (
	"bytes"
	"context"
	"testing"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/test"
)

func TestRun(t *testing.T) {
	var ctx = context.Background()

	// task state is reused throughout the test
	testTasksSQL := `
select initiator, status, operations from tasks order by created_at desc limit 1;
`
	var initiator, status string
	var operations int

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
	for _, v := range []string{"foo/bar.jpg", "bar/foo.jpg", "foo.jpg"} {
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

	var contentString2 = "hello2"
	for _, v := range []string{"foo/bar/baz.jpg"} {
		_, err = minioClient.PutObject(
			ctx,
			"example",
			v,
			bytes.NewReader([]byte(contentString2)),
			int64(len(contentString2)),
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
	if exp, got := 4, report.ObjectsCreated; exp != got {
		t.Fatalf("Expected %d objects to be created, got %d", exp, got)
	}
	if exp, got := 2, report.BlobsCreated; exp != got {
		t.Fatalf("Expected %d blobs to be created, got %d", exp, got)
	}
	if exp, got := 4, report.BlobsLinked; exp != got {
		t.Fatalf("Expected %d blobs to be linked, got %d", exp, got)
	}
	if exp, got := 2, report.ObjectStatCalls; exp != got {
		t.Fatalf("Expected %d object stat calls, got %d", exp, got)
	}

	// check the task state
	err = db.QueryRow(testTasksSQL).Scan(&initiator, &status, &operations)
	if err != nil {
		t.Fatalf("Could not run test tasks SQL: %s", err)
	}

	if exp, got := 12, operations; exp != got {
		t.Fatalf("Expected operations to be %d, got %d", exp, got)
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

	// check task state
	err = db.QueryRow(testTasksSQL).Scan(&initiator, &status, &operations)
	if err != nil {
		t.Fatalf("Could not run test tasks SQL: %s", err)
	}

	if exp, got := 0, operations; exp != got {
		t.Fatalf("Expected operations to be %d, got %d", exp, got)
	}

	// test the state in the database
	testSQL := `set schema 'storage_console';
select
  (select count(id) from blobs) as blob_count,
  (select count(id) from buckets) as bucket_count,
  (select count(id) from content_types) as content_type_count,
  (select count(*) from object_blobs) as object_blob_count,
  (select count(id) from object_storage_providers) as object_storage_provider_count,
  (select count(id) from objects) as object_count;
`

	var blobCount, bucketCount, contentTypeCount, objectBlobCount, objectStorageProviderCount, objectCount int
	err = db.QueryRow(testSQL).Scan(&blobCount, &bucketCount, &contentTypeCount, &objectBlobCount, &objectStorageProviderCount, &objectCount)
	if err != nil {
		t.Fatalf("Could not run test SQL: %s", err)
	}

	if exp, got := 2, blobCount; exp != got {
		t.Fatalf("Expected %d blobs, got %d", exp, got)
	}

	if exp, got := 1, bucketCount; exp != got {
		t.Fatalf("Expected %d bucket, got %d", exp, got)
	}

	if exp, got := 1, contentTypeCount; exp != got {
		t.Fatalf("Expected %d content types, got %d", exp, got)
	}

	if exp, got := 4, objectBlobCount; exp != got {
		t.Fatalf("Expected %d object blobs, got %d", exp, got)
	}

	if exp, got := 1, objectStorageProviderCount; exp != got {
		t.Fatalf("Expected %d object storage providers, got %d", exp, got)
	}

	if exp, got := 4, objectCount; exp != got {
		t.Fatalf("Expected 4 objects, got %d", objectCount)
	}

	testContentsSQL := `
with
  blob as (select * from blobs where md5 = $1),
  objBlob as (select * from object_blobs where blob_id = (select id from blob) and object_id = $2),
  obj as (select * from objects where id = (select object_id from objBlob)),
  bucket as (select * from buckets where id = (select bucket_id from obj))
select (select name from bucket) as bucket, (select key from obj) as obj;`

	var bucket, obj string
	// hello2 blob
	err = db.QueryRow(testContentsSQL, "6e809cbda0732ac4845916a59016f954", 4).Scan(&bucket, &obj)
	if err != nil {
		t.Fatalf("Could not run test contents SQL: %s", err)
	}

	if exp, got := "example", bucket; exp != got {
		t.Fatalf("Expected bucket to be %s, got %s", exp, got)
	}

	if exp, got := "foo/bar/baz.jpg", obj; exp != got {
		t.Fatalf("Expected object key to be %s, got %s", exp, got)
	}

	// update an object
	var newContentString = "hello3"
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
	if exp, got := 0, report.ObjectsCreated; exp != got {
		t.Fatalf("Expected %d objects to be created, got %d", exp, got)
	}
	if exp, got := 1, report.BlobsLinked; exp != got {
		t.Fatalf("Expected %d blobs to be linked, got %d", exp, got)
	}
	if exp, got := 1, report.ObjectStatCalls; exp != got {
		t.Fatalf("Expected %d object stat calls, got %d", exp, got)
	}

	// check task state
	err = db.QueryRow(testTasksSQL).Scan(&initiator, &status, &operations)
	if err != nil {
		t.Fatalf("Could not run test tasks SQL: %s", err)
	}

	if exp, got := 2, operations; exp != got {
		t.Fatalf("Expected operations to be %d, got %d", exp, got)
	}

	// delete an object
	err = minioClient.RemoveObject(ctx, "example", "foo/bar/baz.jpg", minio.RemoveObjectOptions{})
	if err != nil {
		t.Fatalf("Could not remove object: %s", err)
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

	if exp, got := 0, report.ObjectsCreated; exp != got {
		t.Fatalf("Expected %d objects to be created, got %d", exp, got)
	}

	if exp, got := 1, report.ObjectsDeleted; exp != got {
		t.Fatalf("Expected %d objects to be deleted, got %d", exp, got)
	}

	// check task state
	err = db.QueryRow(testTasksSQL).Scan(&initiator, &status, &operations)
	if err != nil {
		t.Fatalf("Could not run test tasks SQL: %s", err)
	}

	if exp, got := "importer", initiator; exp != got {
		t.Fatalf("Expected initiator to be %s, got %s", exp, got)
	}

	if exp, got := "completed", status; exp != got {
		t.Fatalf("Expected status to be %s, got %s", exp, got)
	}

	if exp, got := 1, operations; exp != got {
		t.Fatalf("Expected operations to be %d, got %d", exp, got)
	}
}
