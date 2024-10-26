package thumbnail

import (
	"context"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/importer"
	"github.com/charlieegan3/storage-console/pkg/test"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

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

	err = minioClient.MakeBucket(ctx, "meta", minio.MakeBucketOptions{})
	if err != nil {
		t.Fatalf("Could not create bucket: %s", err)
	}

	files, err := os.ReadDir("fixtures")
	if err != nil {
		t.Fatalf("Could not read fixtures: %s", err)
	}

	for _, f := range files {
		r, err := os.Open("fixtures/" + f.Name())
		if err != nil {
			t.Fatalf("Could not open file: %s", err)
		}

		statData, err := r.Stat()
		if err != nil {
			t.Fatalf("Could not stat file: %s", err)
		}

		_, err = minioClient.PutObject(
			ctx,
			"example",
			"data/"+f.Name(),
			r,
			statData.Size(),
			minio.PutObjectOptions{
				ContentType: "image/jpeg",
			},
		)
		if err != nil {
			t.Fatalf("Could not put object: %s", err)
		}
	}

	// run the importer to set the initial state
	importReport, err := importer.Run(ctx, db, minioClient, &importer.Options{
		BucketName: "example",
		SchemaName: "storage_console",
	})
	if err != nil {
		t.Fatalf("Could not run import: %s", err)
	}
	if importReport.BlobsCreated != 3 {
		t.Fatalf("Expected 3 blobs to be created, got %d", importReport.BlobsCreated)
	}

	thumbReport, err := Run(ctx, db, minioClient, &Options{
		SchemaName: "storage_console",
		BucketName: "example",
	})
	if err != nil {
		t.Fatalf("Could not run thumbnail task: %s", err)
	}

	if thumbReport == nil {
		t.Fatalf("Expected report to be returned")
	}

	if thumbReport.ThumbsCreated != 3 {
		t.Fatalf("Expected 3 thumbs to be created, got %d", thumbReport.ThumbsCreated)
	}
}
