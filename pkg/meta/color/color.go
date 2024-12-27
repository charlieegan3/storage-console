package color

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/charlieegan3/storage-console/pkg/meta"
	"github.com/minio/minio-go/v7"
)

type ColorAnalysisProcessor struct{}

func (c *ColorAnalysisProcessor) Name() string {
	return "color"
}

func (t *ColorAnalysisProcessor) ContentTypes() []string {
	return []string{"image/jpeg", "image/jpg", "image/jp2"}
}

func (c *ColorAnalysisProcessor) Process(
	ctx context.Context,
	objectInfo *minio.ObjectInfo,
	content []byte,
) ([]meta.PutMetadata, error) {
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	colors, err := prominentcolor.Kmeans(img)
	if err != nil {
		return nil, fmt.Errorf("failed to extract prominent colors: %w", err)
	}

	jsonData, err := json.Marshal(colors)
	if err != nil {
		return nil, fmt.Errorf("error converting colors to JSON: %w", err)
	}

	putMetadata := meta.PutMetadata{
		ContentType: meta.JSON,
		Content:     jsonData,
	}

	return []meta.PutMetadata{putMetadata}, nil
}
