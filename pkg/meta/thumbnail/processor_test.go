package thumbnail_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/meta"
	"github.com/charlieegan3/storage-console/pkg/meta/thumbnail"
)

func TestThumbnailProcessor(t *testing.T) {
	imagePath := "../fixtures/rx100-landscape.jpg"
	expectedThumbnailPath := "../fixtures/rx100-landscape-thumbnail.jpg"

	content, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	}

	expectedThumbnail, err := os.ReadFile(expectedThumbnailPath)
	if err != nil {
		t.Fatalf("failed to read expected thumbnail file: %v", err)
	}

	processor := thumbnail.ThumbnailProcessor{MaxSize: 100}

	metadata, err := processor.Process(&minio.ObjectInfo{
		ETag: "foobar",
	}, content)
	if err != nil {
		t.Fatalf("failed to process image: %v", err)
	}

	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata entry, got %d", len(metadata))
	}

	if metadata[0].ContentType != meta.JPG {
		t.Fatalf("expected content type 'jpeg', got '%v'", metadata[0].ContentType)
	}

	if !bytes.Equal(metadata[0].Content, expectedThumbnail) {
		t.Errorf("thumbnail content does not match expected output")
	}
}
