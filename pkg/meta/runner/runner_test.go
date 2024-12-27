package runner_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/database"
	"github.com/charlieegan3/storage-console/pkg/importer"
	"github.com/charlieegan3/storage-console/pkg/meta/runner"
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

	logger := log.New(test.NewTLogWriter(t), "", 0)

	// run the importer to set the initial state
	importReport, err := importer.Run(ctx, db, minioClient, &importer.Options{
		BucketName:  "example",
		SchemaName:  "storage_console",
		LoggerError: logger,
		LoggerInfo:  logger,
	})
	if err != nil {
		t.Fatalf("Could not run import: %s", err)
	}
	if importReport.BlobsCreated != 3 {
		t.Fatalf("Expected 3 blobs to be created, got %d", importReport.BlobsCreated)
	}

	rpt, err := runner.Run(ctx, db, minioClient, &runner.Options{
		BucketName:        "example",
		SchemaName:        "storage_console",
		EnabledProcessors: []string{"thumbnail", "exif", "color"},
		LoggerError:       logger,
		LoggerInfo:        logger,
	})
	if err != nil {
		t.Fatalf("Could not run runner: %s", err)
	}

	if thumbCounts, ok := rpt.Counts["thumbnail"]; !ok || thumbCounts != 3 {
		t.Fatalf("Expected 3 thumbs to be created, got %d", thumbCounts)
	}

	if exifCounts, ok := rpt.Counts["exif"]; !ok || exifCounts != 3 {
		t.Fatalf("Expected 3 exifs to be created, got %d", exifCounts)
	}

	if colorCounts, ok := rpt.Counts["color"]; !ok || colorCounts != 3 {
		t.Fatalf("Expected 3 colors to be created, got %d", colorCounts)
	}

	path := "meta/thumbnail/288e02f05769a83e474a2e961cb52a7a.jpg"
	_, err = minioClient.StatObject(
		ctx,
		"example",
		path,
		minio.StatObjectOptions{},
	)
	if err != nil {
		t.Fatalf("Could not stat object %s: %s", path, err)
	}

	txn, err := database.NewTxnWithSchema(db, "storage_console")
	if err != nil {
		t.Fatalf("Could not start transaction: %s", err)
	}

	row := txn.QueryRow(`select count(*) from blob_metadata where thumbnail = TRUE group by thumbnail;`)

	var count int64
	err = row.Scan(&count)
	if err != nil {
		t.Fatalf("Could not scan: %s", err)
	}

	if count != 3 {
		t.Fatalf("Expected count to be 3, got %d", count)
	}
}
