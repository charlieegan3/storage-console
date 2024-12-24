package exif_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/charlieegan3/storage-console/pkg/meta"
	"github.com/charlieegan3/storage-console/pkg/meta/exif"
	"github.com/minio/minio-go/v7"
)

func TestExifMetadataProcessor(t *testing.T) {
	imagePath := "../fixtures/rx100-landscape.jpg"

	content, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	}

	processor := exif.ExifMetadataProcessor{}

	metadata, err := processor.Process(&minio.ObjectInfo{
		ETag: "foobar",
	}, content)
	if err != nil {
		t.Fatalf("failed to process image: %v", err)
	}

	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata entry, got %d", len(metadata))
	}

	if metadata[0].ContentType != meta.JSON {
		t.Fatalf("expected content type 'json', got '%v'", metadata[0].ContentType)
	}

	expectedMake := "SONY"
	expectedModel := "DSC-RX100M7"
	expectedLensModel := "24-200mm F2.8-4.5"

	var exifData map[string]interface{}
	err = json.Unmarshal(metadata[0].Content, &exifData)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON content: %v", err)
	}

	if makeValue, ok := exifData["Make"].(string); !ok || makeValue != expectedMake {
		t.Errorf("expected Make '%s', got '%v'", expectedMake, makeValue)
	}

	if modelValue, ok := exifData["Model"].(string); !ok || modelValue != expectedModel {
		t.Errorf("expected Model '%s', got '%v'", expectedModel, modelValue)
	}

	if lensModelValue, ok := exifData["LensModel"].(string); !ok || lensModelValue != expectedLensModel {
		t.Errorf("expected LensModel '%s', got '%v'", expectedLensModel, lensModelValue)
	}
}
