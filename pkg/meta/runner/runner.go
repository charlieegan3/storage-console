package runner

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"

	"github.com/charlieegan3/storage-console/pkg/database"
	"github.com/charlieegan3/storage-console/pkg/meta"
	"github.com/charlieegan3/storage-console/pkg/meta/color"
	"github.com/charlieegan3/storage-console/pkg/meta/exif"
	"github.com/charlieegan3/storage-console/pkg/meta/thumbnail"
	"github.com/minio/minio-go/v7"
)

//go:embed needs_metadatas.sql
var needsMetadatasSQL string

const (
	metaPath = "meta/"
	dataPath = "data/"
)

type Report struct {
	Counts map[string]int
}

type Options struct {
	SchemaName string
	BucketName string

	Prefix string

	EnabledProcessors []string

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}

type blob struct {
	ID  int
	MD5 string
	Key string
}

func Run(
	ctx context.Context,
	db *sql.DB,
	minioClient *minio.Client,
	opts *Options,
) (*Report, error) {
	var processors []meta.MetadataOperationProcessor
	for _, processorName := range opts.EnabledProcessors {
		processor, err := processorForName(processorName)
		if err != nil {
			return nil, fmt.Errorf("could not get processor: %s", err)
		}

		processors = append(processors, processor)
	}

	txn, err := database.NewTxnWithSchema(db, opts.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %s", err)
	}

	var rpt Report
	rpt.Counts = make(map[string]int)

	blobProcessors := make(map[string][]string)
	blobs := make(map[string]blob)

	for _, processor := range processors {
		contentTypes := "'" + strings.Join(processor.ContentTypes(), "', '") + "'"

		rows, err := txn.QueryContext(
			ctx,
			fmt.Sprintf(needsMetadatasSQL, processor.Name(), processor.Name(), contentTypes),
		)
		if err != nil {
			return nil, fmt.Errorf("could not select missing blobs: %s", err)
		}

		for rows.Next() {
			var blob blob
			err = rows.Scan(&blob.ID, &blob.MD5, &blob.Key)
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("could not scan path: %s", err)
			}

			if !strings.HasPrefix(blob.Key, opts.Prefix) {
				continue
			}

			blobProcessors[blob.MD5] = append(blobProcessors[blob.MD5], processor.Name())
			blobs[blob.MD5] = blob
		}
	}

	var putMetadatas []meta.PutMetadata
	for _, blob := range blobs {
		obj, err := minioClient.GetObject(
			ctx,
			opts.BucketName,
			path.Join(dataPath, blob.Key),
			minio.GetObjectOptions{},
		)
		if err != nil {
			return nil, fmt.Errorf("could not get object %s: %s", blob.Key, err)
		}

		objStat, err := obj.Stat()
		if err != nil {
			return nil, fmt.Errorf("could not stat object %s: %s", blob.Key, err)
		}

		bs, err := io.ReadAll(obj)
		if err != nil {
			return nil, fmt.Errorf("could not read object: %s", err)
		}

		opts.LoggerInfo.Printf("processing metadata for blob %s", blob.Key)

		for _, processorName := range blobProcessors[blob.MD5] {
			processor, err := processorForName(processorName)
			if err != nil {
				return nil, fmt.Errorf("could not get processor: %s", err)
			}

			pms, err := processor.Process(ctx, &objStat, bs)
			if err != nil {
				return nil, fmt.Errorf("could not process blob: %s", err)
			}

			result := "unknown"
			if len(pms) > 0 {
				result = "success"
			} else {
				result = "failure"
			}

			setMetaSQL := `
INSERT INTO blob_metadata (blob_id, %s)
VALUES ($1, $2)
ON CONFLICT (blob_id)
DO UPDATE SET %s = $2;`

			_, err = txn.Exec(fmt.Sprintf(setMetaSQL, processor.Name(), processor.Name()), blob.ID, result)
			if err != nil {
				return nil, fmt.Errorf("could not set metadata: %s", err)
			}

			rpt.Counts[processorName] += len(pms)

			putMetadatas = append(putMetadatas, pms...)
		}
	}

	for _, putMetadata := range putMetadatas {
		if putMetadata.Path == "" {
			return nil, fmt.Errorf("metadata path must be set")
		}

		_, err := minioClient.PutObject(
			ctx,
			opts.BucketName,
			path.Join(metaPath, putMetadata.Path),
			bytes.NewReader(putMetadata.Content),
			int64(len(putMetadata.Content)),
			minio.PutObjectOptions{
				ContentType: meta.ContentTypeToString(putMetadata.ContentType),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("could not put metadata: %s", err)
		}
	}

	err = txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %s", err)
	}

	return &rpt, nil
}

func processorForName(name string) (meta.MetadataOperationProcessor, error) {
	switch name {
	case "thumbnail":
		return &thumbnail.ThumbnailProcessor{
			MaxSize: 300,
		}, nil
	case "color":
		return &color.ColorAnalysisProcessor{}, nil
	case "exif":
		return &exif.ExifMetadataProcessor{}, nil
	default:
		return nil, fmt.Errorf("unknown processor: %s", name)
	}
}
