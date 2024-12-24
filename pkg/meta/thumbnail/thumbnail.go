package thumbnail

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
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

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}

func Run(
	ctx context.Context,
	db *sql.DB,
	minioClient *minio.Client,
	opts *Options,
) (*Report, error) {
	vips.LoggingSettings(func(messageDomain string, messageLevel vips.LogLevel, message string) {}, vips.LogLevelCritical)

	var rep Report

	if opts.ThumbMaxSize <= 0 {
		return nil, fmt.Errorf("invalid ThumbMaxSize: %d", opts.ThumbMaxSize)
	}

	missingThumbs, err := fetchMissingThumbs(db, opts.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("could not get missing thumbs: %w", err)
	}

	for _, thumb := range missingThumbs {
		o, err := minioClient.GetObject(
			ctx,
			opts.BucketName,
			path.Join(dataPath, thumb.key),
			minio.GetObjectOptions{},
		)
		if err != nil {
			opts.LoggerError.Printf("could not get object: %s", err)

			continue
		}

		stat, err := o.Stat()
		if err != nil || stat.ETag != thumb.md5 {
			opts.LoggerError.Printf("md5 mismatch or error for key %s: %v", thumb.key, err)

			continue
		}

		thumbReader, size, err := processThumbnail(o, opts.ThumbMaxSize)
		if err != nil {
			opts.LoggerError.Printf("could not process image %s: %v", thumb.key, err)

			continue
		}

		_, err = minioClient.PutObject(
			ctx,
			opts.BucketName,
			path.Join(metaPath, "thumbnail", thumb.md5+".jpg"),
			thumbReader,
			size,
			minio.PutObjectOptions{
				ContentType: "image/jpeg",
			},
		)
		if err != nil {
			opts.LoggerError.Printf("could not put thumbnail: %v", err)

			continue
		}

		err = setThumbnail(db, opts.SchemaName, thumb.id)
		if err != nil {
			opts.LoggerError.Printf("could not set thumb for ID %d: %v", thumb.id, err)

			continue
		}

		rep.ThumbsCreated++

		opts.LoggerInfo.Printf("thumbnail created (%d/%d): %s", rep.ThumbsCreated, len(missingThumbs), thumb.key)
	}

	return &rep, nil
}

func fetchMissingThumbs(db *sql.DB, schemaName string) ([]struct {
	id       int
	md5, key string
}, error,
) {
	txn, err := database.NewTxnWithSchema(db, schemaName)
	if err != nil {
		return nil, fmt.Errorf("could not create transaction: %w", err)
	}
	defer txn.Rollback()

	rows, err := txn.Query(missingThumbsSQL)
	if err != nil {
		return nil, fmt.Errorf("could not query missing thumbs: %w", err)
	}
	defer rows.Close()

	var thumbs []struct {
		id       int
		md5, key string
	}
	for rows.Next() {
		var thumb struct {
			id       int
			md5, key string
		}
		if err := rows.Scan(&thumb.id, &thumb.md5, &thumb.key); err != nil {
			return nil, fmt.Errorf("could not scan row: %w", err)
		}
		thumbs = append(thumbs, thumb)
	}

	return thumbs, nil
}

func setThumbnail(db *sql.DB, schemaName string, thumbID int) error {
	txn, err := database.NewTxnWithSchema(db, schemaName)
	if err != nil {
		return fmt.Errorf("could not create transaction: %w", err)
	}
	defer txn.Rollback()

	_, err = txn.Exec(setThumbSQL, thumbID)
	if err != nil {
		return fmt.Errorf("could not set thumb: %w", err)
	}

	return txn.Commit()
}

func processThumbnail(reader io.Reader, maxSize int) (io.Reader, int64, error) {
	if maxSize <= 0 {
		return nil, 0, fmt.Errorf("invalid maxSize: %d", maxSize)
	}

	originalImage, err := vips.NewImageFromReader(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("could not load image: %w", err)
	}

	longestSide := originalImage.Width()
	if originalImage.Height() > originalImage.Width() {
		longestSide = originalImage.Height()
	}

	fmt.Println(longestSide)

	if longestSide > maxSize {
		if longestSide <= 0 {
			return nil, 0, fmt.Errorf("invalid longestSide: %d", longestSide)
		}

		scale := float64(maxSize) / float64(longestSide)
		if scale <= 0 {
			return nil, 0, fmt.Errorf("invalid scaling factor: %f", scale)
		}

		if err := originalImage.Resize(scale, vips.KernelNearest); err != nil {
			return nil, 0, fmt.Errorf("could not resize image: %w", err)
		}
	}

	if err := originalImage.AutoRotate(); err != nil {
		return nil, 0, fmt.Errorf("could not rotate image: %w", err)
	}

	ep := vips.NewDefaultJPEGExportParams()
	thumbBytes, _, err := originalImage.Export(ep)
	if err != nil {
		return nil, 0, fmt.Errorf("could not export image: %w", err)
	}

	return bytes.NewBuffer(thumbBytes), int64(len(thumbBytes)), nil
}
