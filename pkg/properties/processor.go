package properties

import (
	"context"
	"fmt"
	"strings"
	"time"
)

var PredefinedColors = map[string]Color{
	"r":  {255, 0, 0},
	"g":  {0, 255, 0},
	"b":  {0, 0, 255},
	"c":  {0, 255, 255},
	"m":  {255, 0, 255},
	"y":  {255, 255, 0},
	"ro": {255, 127, 0},
	"yo": {255, 191, 0},
	"yg": {127, 255, 0},
	"bg": {0, 255, 127},
	"bv": {0, 127, 255},
	"rv": {127, 0, 255},
}

type Color struct {
	R int `json:"R"`
	G int `json:"G"`
	B int `json:"B"`
}

type Processor interface {
	Name() string
	Process(ctx context.Context, content []byte) ([]BlobProperties, error)
}

type BlobProperties struct {
	BlobID int `db:"blob_id"`

	PropertySource string `db:"source"`

	PropertyType string `db:"property_type"`

	ValueType string `db:"value_type"`

	ValueBool        *bool      `db:"value_bool"`
	ValueText        *string    `db:"value_text"`
	ValueNumerator   *int       `db:"value_numerator"`
	ValueDenominator *int       `db:"value_denominator"`
	ValueInteger     *int       `db:"value_integer"`
	ValueFloat       *float64   `db:"value_float"`
	ValueTimestamp   *time.Time `db:"value_timestamp"`
	ValueTimestamptz *time.Time `db:"value_timestamptz"`
}

func (bp *BlobProperties) Color() string {
	if bp.PropertySource != "color" {
		return ""
	}

	if bp.ValueText == nil {
		return ""
	}

	if strings.HasPrefix(bp.PropertyType, "ColorCategory") {
		c := PredefinedColors[*bp.ValueText]
		return fmt.Sprintf("%d,%d,%d", c.R, c.G, c.B)
	}

	return *bp.ValueText
}

func (bp *BlobProperties) String() string {
	switch {
	case bp.ValueType == "Bool" && bp.ValueBool != nil:
		if *bp.ValueBool {
			return "True"
		}

		return "False"
	case bp.ValueType == "Text" && bp.ValueText != nil:
		return *bp.ValueText
	case bp.ValueType == "Integer" && bp.ValueInteger != nil:
		return fmt.Sprintf("%d", *bp.ValueInteger)
	case bp.ValueType == "Float" && bp.ValueFloat != nil:
		return fmt.Sprintf("%g", *bp.ValueFloat)
	case bp.ValueType == "Timestamp" && bp.ValueTimestamp != nil:
		return fmt.Sprintf("%v", *bp.ValueTimestamp)
	case bp.ValueType == "TimestampWithTimeZone" && bp.ValueTimestamptz != nil:
		return fmt.Sprintf("%v", *bp.ValueTimestamptz)
	case bp.ValueType == "Fraction" && bp.ValueNumerator != nil && bp.ValueDenominator != nil:
		if *bp.ValueDenominator == 1 {
			return fmt.Sprintf("%d", *bp.ValueNumerator)
		}

		if *bp.ValueNumerator == 0 {
			return "0"
		}

		return fmt.Sprintf("%d/%d", *bp.ValueNumerator, *bp.ValueDenominator)
	}

	return "Unknown"
}
