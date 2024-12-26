package color

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"math"
	"strings"

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

type ClassifiedColor struct {
	Hex      string                  `json:"hex"`
	Name     string                  `json:"name"`
	Count    int                     `json:"count"`
	Original prominentcolor.ColorRGB `json:"original_rgb"`
}

// Predefined colors for classification
var predefinedColors = []struct {
	Name string
	R    uint32
	G    uint32
	B    uint32
}{
	{"black", 0, 0, 0},
	{"white", 255, 255, 255},
	{"red", 255, 0, 0},
	{"yellow", 255, 255, 0},
	{"blue", 0, 0, 255},
	{"orange", 255, 165, 0},
	{"green", 0, 128, 0},
	{"purple", 128, 0, 128},
	{"red-orange", 255, 69, 0},
	{"yellow-orange", 255, 200, 0},
	{"yellow-green", 154, 205, 50},
	{"blue-green", 0, 128, 128},
	{"blue-purple", 138, 43, 226},
	{"red-purple", 199, 21, 133},
}

// Calculate Euclidean distance between two colors
func colorDistance(r1, g1, b1, r2, g2, b2 uint32) float64 {
	return math.Sqrt(float64((r1-r2)*(r1-r2) + (g1-g2)*(g1-g2) + (b1-b2)*(b1-b2)))
}

// Classify a color to the closest predefined color
func classifyColor(r, g, b uint32) string {
	minDistance := math.MaxFloat64
	closestColor := "unknown"

	for _, color := range predefinedColors {
		distance := colorDistance(r, g, b, color.R, color.G, color.B)
		if distance < minDistance {
			minDistance = distance
			closestColor = color.Name
		}
	}

	return closestColor
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

	var classifiedColors []ClassifiedColor
	for _, color := range colors {
		r := color.Color.R
		g := color.Color.G
		b := color.Color.B

		classifiedColors = append(classifiedColors, ClassifiedColor{
			Hex:      strings.ToUpper(fmt.Sprintf("#%02X%02X%02X", r, g, b)),
			Name:     classifyColor(r, g, b),
			Count:    color.Cnt,
			Original: color.Color,
		})
	}

	jsonData, err := json.Marshal(classifiedColors)
	if err != nil {
		return nil, fmt.Errorf("error converting classified colors to JSON: %w", err)
	}

	putMetadata := meta.PutMetadata{
		ContentType: meta.JSON,
		Content:     jsonData,
	}

	return []meta.PutMetadata{putMetadata}, nil
}
