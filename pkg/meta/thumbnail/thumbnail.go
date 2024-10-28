package thumbnail

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"path"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/database"
)

const (
	metaPath = "meta/"
	dataPath = "data/"
)

//go:embed missing_thumbs.sql
var missingThumbsSQL string

//go:embed set_thumb.sql
var setThumbSQL string

type Report struct {
	ThumbsCreated int
}

type Options struct {
	ThumbMaxSize int
	SchemaName   string
	BucketName   string
}

func Run(
	ctx context.Context,
	db *sql.DB,
	minioClient *minio.Client,
	opts *Options,
) (*Report, error) {
	var success bool
	txn, err := database.NewTxnWithSchema(db, opts.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("could not create transaction: %w", err)
	}
	defer func() {
		if !success {
			err := txn.Rollback()
			if err != nil {
				log.Printf("could not rollback transaction: %s", err)
			}
		}
	}()

	rows, err := txn.Query(missingThumbsSQL)
	if err != nil {
		return nil, fmt.Errorf("could not get missing thumbs: %w", err)
	}

	var missingThumbs []struct {
		id  int
		md5 string
		key string
	}
	for rows.Next() {
		var thumb struct {
			id  int
			md5 string
			key string
		}
		err = rows.Scan(&thumb.id, &thumb.md5, &thumb.key)
		if err != nil {
			return nil, fmt.Errorf("could not scan missing thumb: %w", err)
		}
		missingThumbs = append(missingThumbs, thumb)
	}

	var rep Report
	for _, thumb := range missingThumbs {
		o, err := minioClient.GetObject(
			ctx,
			opts.BucketName,
			path.Join(dataPath, thumb.key),
			minio.GetObjectOptions{},
		)
		if err != nil {
			return nil, fmt.Errorf("could not get object: %w", err)
		}

		stat, err := o.Stat()
		if err != nil {
			return nil, fmt.Errorf("could not stat object: %w", err)
		}
		if stat.ETag != thumb.md5 {
			log.Println("md5 mismatch", thumb.key, thumb.md5, stat.ETag)
			continue
		}

		originalImage, err := vips.NewImageFromReader(o)
		if err != nil {
			return nil, fmt.Errorf("could not load image %s: %w", thumb.key, err)
		}

		longestSide := originalImage.Width()
		if originalImage.Height() > originalImage.Width() {
			longestSide = originalImage.Height()
		}

		if longestSide > opts.ThumbMaxSize {
			err := originalImage.Resize(float64(opts.ThumbMaxSize)/float64(longestSide), vips.KernelNearest)
			if err != nil {
				return nil, fmt.Errorf("could not resize image: %w", err)
			}
		}

		err = originalImage.AutoRotate()
		if err != nil {
			return nil, fmt.Errorf("could not rotate image: %w", err)
		}

		ep := vips.NewDefaultJPEGExportParams()
		thumbBytes, _, err := originalImage.Export(ep)
		if err != nil {
			return nil, fmt.Errorf("could not export image: %w", err)
		}

		_, err = minioClient.PutObject(
			ctx,
			opts.BucketName,
			path.Join(metaPath, "thumbnail", thumb.md5+".jpg"),
			bytes.NewBuffer(thumbBytes),
			int64(len(thumbBytes)),
			minio.PutObjectOptions{
				ContentType: "image/jpeg",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("could not put thumbnail: %w", err)
		}

		_, err = txn.Exec(setThumbSQL, thumb.id)
		if err != nil {
			return nil, fmt.Errorf("could not set thumb: %w", err)
		}

		rep.ThumbsCreated++
	}

	err = txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	success = true

	return &rep, nil
}

func getSize(a, b, c int) int {
	d := a * b / c
	return (d + 1) & -1
}
