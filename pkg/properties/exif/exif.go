package exif

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/charlieegan3/storage-console/pkg/properties"
)

type ExifProcessor struct{}

func (e *ExifProcessor) Name() string {
	return "exif"
}

const source = "exif"

func (e *ExifProcessor) Process(
	ctx context.Context,
	content []byte,
) ([]properties.BlobProperties, error) {
	var props []properties.BlobProperties
	var em exifMetadata

	err := json.Unmarshal(content, &em)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal exif metadata: %w", err)
	}

	// Extract ApertureValue
	if len(em.ApertureValue) > 0 {
		value := float64(em.ApertureValue[0].Numerator) / float64(em.ApertureValue[0].Denominator)
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "ApertureValue",
			ValueType:      "Float",
			ValueFloat:     []*float64{&value}[0],
		})
	}

	// Extract ExposureBiasValue
	if len(em.ExposureBiasValue) > 0 {
		props = append(props, properties.BlobProperties{
			PropertySource:   source,
			PropertyType:     "ExposureBiasValue",
			ValueType:        "Fraction",
			ValueNumerator:   &em.ExposureBiasValue[0].Numerator,
			ValueDenominator: &em.ExposureBiasValue[0].Denominator,
		})
	}

	// Extract GPSAltitude
	if len(em.GPSAltitude) > 0 {
		value := float64(em.GPSAltitude[0].Numerator) / float64(em.GPSAltitude[0].Denominator)
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "GPSAltitude",
			ValueType:      "Integer",
			ValueInteger:   &[]int{int(value)}[0],
		})
	}

	// Extract Make
	if em.Make != "" {
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "Make",
			ValueType:      "Text",
			ValueText:      &em.Make,
		})
	}

	// Extract Model
	if em.Model != "" {
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "Model",
			ValueType:      "Text",
			ValueText:      &em.Model,
		})
	}

	// Extract Software
	if em.Software != "" {
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "Software",
			ValueType:      "Text",
			ValueText:      &em.Software,
		})
	}

	// Extract DateTimeOriginal
	if em.DateTimeOriginal != "" {
		timestamp, err := time.Parse("2006:01:02 15:04:05", em.DateTimeOriginal)
		if err == nil {
			props = append(props, properties.BlobProperties{
				PropertySource: source,
				PropertyType:   "DateTimeOriginal",
				ValueType:      "Timestamp",
				ValueTimestamp: &timestamp,
			})
		}
	}

	// Extract OffsetTimeOriginal
	if em.OffsetTimeOriginal != "" {
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "OffsetTimeOriginal",
			ValueType:      "Text",
			ValueText:      &em.OffsetTimeOriginal,
		})
	}

	// Extract ExposureTime
	if len(em.ExposureTime) > 0 {
		props = append(props, properties.BlobProperties{
			PropertySource:   source,
			PropertyType:     "ExposureTime",
			ValueType:        "Fraction",
			ValueNumerator:   &em.ExposureTime[0].Numerator,
			ValueDenominator: &em.ExposureTime[0].Denominator,
		})
	}

	// Extract ISOSpeedRatings
	if len(em.ISOSpeedRatings) > 0 {
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "ISOSpeedRatings",
			ValueType:      "Integer",
			ValueInteger:   &em.ISOSpeedRatings[0],
		})
	}

	// Extract LensModel
	if em.LensModel != "" {
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "LensModel",
			ValueType:      "Text",
			ValueText:      &em.LensModel,
		})
	}

	// Extract GPSLatitude and GPSLongitude
	if len(em.GPSLatitude) > 0 && em.GPSLatitudeRef != "" {
		latitude := convertDegrees(em.GPSLatitude, em.GPSLatitudeRef)
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "GPSLatitude",
			ValueType:      "Float",
			ValueFloat:     &latitude,
		})
	}

	if len(em.GPSLongitude) > 0 && em.GPSLongitudeRef != "" {
		longitude := convertDegrees(em.GPSLongitude, em.GPSLongitudeRef)
		props = append(props, properties.BlobProperties{
			PropertySource: source,
			PropertyType:   "GPSLongitude",
			ValueType:      "Float",
			ValueFloat:     &longitude,
		})
	}

	return props, nil
}

// convertDegrees converts GPS coordinates from fractional degrees to decimal format.
func convertDegrees(coord []struct {
	Numerator   int `json:"Numerator"`
	Denominator int `json:"Denominator"`
}, ref string,
) float64 {
	if len(coord) < 3 {
		return 0
	}
	degrees := float64(coord[0].Numerator) / float64(coord[0].Denominator)
	minutes := float64(coord[1].Numerator) / float64(coord[1].Denominator)
	seconds := float64(coord[2].Numerator) / float64(coord[2].Denominator)
	decimal := degrees + (minutes / 60) + (seconds / 3600)
	if ref == "S" || ref == "W" {
		decimal = -decimal
	}
	return decimal
}

type exifMetadata struct {
	ApertureValue []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"ApertureValue"`
	BrightnessValue []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"BrightnessValue"`
	ColorSpace        []int  `json:"ColorSpace"`
	Contrast          []int  `json:"Contrast"`
	CustomRendered    []int  `json:"CustomRendered"`
	DateTime          string `json:"DateTime"`
	DateTimeDigitized string `json:"DateTimeDigitized"`
	DateTimeOriginal  string `json:"DateTimeOriginal"`
	DigitalZoomRatio  []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"DigitalZoomRatio"`
	ExifVersion struct {
		ExifVersion string `json:"ExifVersion"`
	} `json:"ExifVersion"`
	ExposureBiasValue []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"ExposureBiasValue"`
	ExposureMode    []int `json:"ExposureMode"`
	ExposureProgram []int `json:"ExposureProgram"`
	ExposureTime    []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"ExposureTime"`
	FNumber []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"FNumber"`
	Flash       []int `json:"Flash"`
	FocalLength []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"FocalLength"`
	FocalLengthIn35MmFilm    []int `json:"FocalLengthIn35mmFilm"`
	FocalPlaneResolutionUnit []int `json:"FocalPlaneResolutionUnit"`
	FocalPlaneXResolution    []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"FocalPlaneXResolution"`
	FocalPlaneYResolution []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"FocalPlaneYResolution"`
	GPSAltitude []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"GPSAltitude"`
	GPSLatitude []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"GPSLatitude"`
	GPSLatitudeRef string `json:"GPSLatitudeRef"`
	GPSLongitude   []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"GPSLongitude"`
	GPSLongitudeRef   string `json:"GPSLongitudeRef"`
	ISOSpeedRatings   []int  `json:"ISOSpeedRatings"`
	LensModel         string `json:"LensModel"`
	LensSpecification []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"LensSpecification"`
	LightSource      []int  `json:"LightSource"`
	Make             string `json:"Make"`
	MaxApertureValue []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"MaxApertureValue"`
	MeteringMode             []int  `json:"MeteringMode"`
	Model                    string `json:"Model"`
	OffsetTime               string `json:"OffsetTime"`
	OffsetTimeDigitized      string `json:"OffsetTimeDigitized"`
	OffsetTimeOriginal       string `json:"OffsetTimeOriginal"`
	RecommendedExposureIndex []int  `json:"RecommendedExposureIndex"`
	ResolutionUnit           []int  `json:"ResolutionUnit"`
	Saturation               []int  `json:"Saturation"`
	SceneCaptureType         []int  `json:"SceneCaptureType"`
	SensitivityType          []int  `json:"SensitivityType"`
	Sharpness                []int  `json:"Sharpness"`
	ShutterSpeedValue        []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"ShutterSpeedValue"`
	Software           string `json:"Software"`
	SubSecTimeOriginal string `json:"SubSecTimeOriginal"`
	WhiteBalance       []int  `json:"WhiteBalance"`
	XResolution        []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"XResolution"`
	YResolution []struct {
		Numerator   int `json:"Numerator"`
		Denominator int `json:"Denominator"`
	} `json:"YResolution"`
}
