package color_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/charlieegan3/storage-console/pkg/meta/color"
	"github.com/minio/minio-go/v7"
)

func TestColorAnalysisProcessor(t *testing.T) {
	imagePath := "../fixtures/rx100-landscape.jpg"

	content, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	}

	processor := color.ColorAnalysisProcessor{}

	metadata, err := processor.Process(minio.ObjectInfo{
		ETag: "foobar",
	}, content)
	if err != nil {
		t.Fatalf("failed to process image: %v", err)
	}

	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata entry, got %d", len(metadata))
	}

	if metadata[0].Path != "meta/color_analysis/foobar.json" {
		t.Fatalf("expected path 'meta/color_analysis/foobar.json', got '%s'", metadata[0].Path)
	}

	if metadata[0].ContentType != "application/json" {
		t.Errorf("expected content type 'application/json', got '%s'", metadata[0].ContentType)
	}

	var colorData []color.ClassifiedColor
	err = json.Unmarshal(metadata[0].Content, &colorData)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON content: %v", err)
	}

	// Define expected output (based on the input image used)
	expected := []color.ClassifiedColor{
		{
			Hex:   "#CFCEC4",
			Name:  "white",
			Count: 1585,
			Original: prominentcolor.ColorRGB{
				R: 207,
				G: 206,
				B: 196,
			},
		},
		{
			Hex:   "#3E3E20",
			Name:  "black",
			Count: 1480,
			Original: prominentcolor.ColorRGB{
				R: 62,
				G: 62,
				B: 32,
			},
		},
		{
			Hex:   "#84802C",
			Name:  "yellow-green",
			Count: 1175,
			Original: prominentcolor.ColorRGB{
				R: 132,
				G: 128,
				B: 44,
			},
		},
	}

	if len(colorData) != len(expected) {
		t.Fatalf("expected %d colors, got %d", len(expected), len(colorData))
	}

	for i, expectedColor := range expected {
		if colorData[i] != expectedColor {
			t.Errorf("color mismatch at index %d: expected %+v, got %+v", i, expectedColor, colorData[i])
		}
	}
}
