package color

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/charlieegan3/storage-console/pkg/properties"
)

type ColorProcessor struct{}

func (e *ColorProcessor) Name() string {
	return "color"
}

func (e *ColorProcessor) Process(
	ctx context.Context,
	content []byte,
) ([]properties.BlobProperties, error) {
	var props []properties.BlobProperties

	var rawColors []colorData
	err := json.Unmarshal(content, &rawColors)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal colors: %v", err)
	}

	// add the raw colors
	for i, v := range rawColors {
		if i > 2 {
			break
		}

		props = append(props, properties.BlobProperties{
			ValueType:      "Text",
			ValueText:      &[]string{fmt.Sprintf("%d,%d,%d", v.Color.R, v.Color.G, v.Color.B)}[0],
			PropertyType:   fmt.Sprintf("ProminentColor%d", i+1),
			PropertySource: "color",
		})
	}

	for i, v := range mapColors(rawColors) {
		if i > 2 {
			break
		}

		props = append(props, properties.BlobProperties{
			ValueType:      "Text",
			ValueText:      &v,
			PropertyType:   fmt.Sprintf("ColorCategory%d", i+1),
			PropertySource: "color",
		})
	}

	return props, nil
}

type colorData struct {
	Color properties.Color `json:"Color"`
	Cnt   int              `json:"Cnt"`
}

func distance(c1, c2 properties.Color) float64 {
	return math.Sqrt(
		math.Pow(float64(c1.R-c2.R), 2) +
			math.Pow(float64(c1.G-c2.G), 2) +
			math.Pow(float64(c1.B-c2.B), 2),
	)
}

func findNearestColor(c properties.Color) string {
	minDist := math.MaxFloat64
	nearestColor := ""

	for name, predefined := range properties.PredefinedColors {
		dist := distance(c, predefined)
		if dist < minDist {
			minDist = dist
			nearestColor = name
		}
	}

	return nearestColor
}

func mapColors(colors []colorData) []string {
	result := make(map[string]struct{})
	orderedColors := []string{}

	for _, data := range colors {
		nc := findNearestColor(data.Color)
		if _, ok := result[nc]; !ok {
			orderedColors = append(orderedColors, nc)
		}
	}

	return orderedColors
}
