package color_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/meta"
	"github.com/charlieegan3/storage-console/pkg/meta/color"
)

func TestColorAnalysisProcessor(t *testing.T) {
	imagePath := "../fixtures/rx100-landscape.jpg"

	content, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	}

	processor := color.ColorAnalysisProcessor{}

	metadata, err := processor.Process(minio.ObjectInfo{}, content)
	if err != nil {
		t.Fatalf("failed to process image: %v", err)
	}

	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata entry, got %d", len(metadata))
	}

	if metadata[0].ContentType != meta.JSON {
		t.Fatalf("expected content type 'json', got '%v'", metadata[0].ContentType)
	}

	var colorData []prominentcolor.ColorItem
	err = json.Unmarshal(metadata[0].Content, &colorData)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON content: %v", err)
	}

	if len(colorData) != 3 {
		t.Fatalf("expected 3 color entries, got %d", len(colorData))
	}
}
