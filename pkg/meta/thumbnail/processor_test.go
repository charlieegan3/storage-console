package thumbnail_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/charlieegan3/storage-console/pkg/meta/thumbnail"
	"github.com/minio/minio-go/v7"
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

	if metadata[0].Path != "meta/thumbnail/foobar.jpg" {
		t.Fatalf("expected path 'meta/thumbnail/foobar.jpg', got '%s'", metadata[0].Path)
	}

	if metadata[0].ContentType != "image/jpeg" {
		t.Errorf("expected content type 'image/jpeg', got '%s'", metadata[0].ContentType)
	}

	if !bytes.Equal(metadata[0].Content, expectedThumbnail) {
		t.Errorf("thumbnail content does not match expected output")
	}
}
