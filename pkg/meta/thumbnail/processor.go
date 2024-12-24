package thumbnail

import (
	"fmt"

	"github.com/charlieegan3/storage-console/pkg/meta"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/minio/minio-go/v7"
)

type ThumbnailProcessor struct {
	MaxSize int
}

func (p *ThumbnailProcessor) Process(objectInfo *minio.ObjectInfo, content []byte) ([]meta.PutMetadata, error) {
	vips.Startup(nil)
	defer vips.Shutdown()

	image, err := vips.NewImageFromBuffer(content)
	if err != nil {
		return nil, fmt.Errorf("could not load image: %w", err)
	}
	defer image.Close()

	if err := image.AutoRotate(); err != nil {
		return nil, fmt.Errorf("could not auto-rotate image: %w", err)
	}

	width := image.Width()
	height := image.Height()
	longestSide := width
	if height > width {
		longestSide = height
	}

	if longestSide > p.MaxSize {
		scale := float64(p.MaxSize) / float64(longestSide)
		if err := image.Resize(scale, vips.KernelLanczos3); err != nil {
			return nil, fmt.Errorf("could not resize image: %w", err)
		}
	}

	exportParams := vips.NewDefaultJPEGExportParams()
	thumbnailBytes, _, err := image.Export(exportParams)
	if err != nil {
		return nil, fmt.Errorf("could not export thumbnail: %w", err)
	}

	thumbnailPath := fmt.Sprintf("meta/thumbnail/%s.jpg", objectInfo.ETag)
	putMetadata := meta.PutMetadata{
		Path: thumbnailPath,
		PutObjectOptions: minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
		Content: thumbnailBytes,
	}

	return []meta.PutMetadata{putMetadata}, nil
}
