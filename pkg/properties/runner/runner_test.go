package runner_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/importer"
	metaRunner "github.com/charlieegan3/storage-console/pkg/meta/runner"
	"github.com/charlieegan3/storage-console/pkg/properties/runner"
	"github.com/charlieegan3/storage-console/pkg/test"
)

func TestRun(t *testing.T) {
	t.Parallel()
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

	_, err = metaRunner.Run(ctx, db, minioClient, &metaRunner.Options{
		BucketName: "example",
		SchemaName: "storage_console",
		// only need these two for properties
		EnabledProcessors: []string{"exif", "color"},
		LoggerError:       logger,
		LoggerInfo:        logger,
	})
	if err != nil {
		t.Fatalf("Could not run meta runner: %s", err)
	}

	rpt, err := runner.Run(ctx, db, minioClient, &runner.Options{
		BucketName:        "example",
		SchemaName:        "storage_console",
		EnabledProcessors: []string{"exif", "color"},
		LoggerError:       logger,
		LoggerInfo:        logger,
	})
	if err != nil {
		t.Fatalf("Could not run meta runner: %s", err)
	}

	if exp, got := 39, rpt.Counts["exif"]; exp != got {
		t.Fatalf("Expected %d exif properties to be created, got %d", exp, got)
	}

	if exp, got := 18, rpt.Counts["color"]; exp != got {
		t.Fatalf("Expected %d color properties to be created, got %d", exp, got)
	}
}
