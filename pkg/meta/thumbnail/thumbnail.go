package thumbnail

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"path"

	"github.com/charlieegan3/storage-console/pkg/database"
	"github.com/minio/minio-go/v7"
	"golang.org/x/image/draw"
)

const metaPath = "meta/"

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
			thumb.key,
			minio.GetObjectOptions{},
		)
		if err != nil {
			return nil, fmt.Errorf("could not get object: %w", err)
		}

		stat, err := o.Stat()
		if err != nil {
			return nil, fmt.Errorf("could not get object stat: %w", err)
		}
		if stat.ETag != thumb.md5 {
			log.Println("md5 mismatch", thumb.key, thumb.md5, stat.ETag)
			continue
		}

		var origImage image.Image
		origImage, _, err = image.Decode(o)
		if err != nil {
			return nil, fmt.Errorf("could not decode image: %w", err)
		}

		p := origImage.Bounds().Size()
		w, h := opts.ThumbMaxSize, getSize(opts.ThumbMaxSize, p.Y, p.X)
		if p.X < p.Y {
			w, h = getSize(opts.ThumbMaxSize, p.X, p.Y), opts.ThumbMaxSize
		}
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), origImage, origImage.Bounds(), draw.Src, nil)

		b := bytes.NewBuffer(nil)
		err = jpeg.Encode(b, dst, nil)
		if err != nil {
			return nil, fmt.Errorf("could not encode thumbnail: %w", err)
		}

		_, err = minioClient.PutObject(
			ctx,
			opts.BucketName,
			path.Join(metaPath, "thumbnail", thumb.md5+".jpg"),
			b,
			int64(b.Len()),
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
