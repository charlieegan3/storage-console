package properties

import (
	"context"
	"time"
)

type BlobProperties struct {
	BlobID int `db:"blob_id"`

	PropertySource string `db:"source"`

	PropertyType string `db:"property_type"`

	ValueType string `db:"value_type"`

	ValueBool        *bool      `db:"value_bool"`
	ValueNumerator   *int       `db:"value_numerator"`
	ValueDenominator *int       `db:"value_denominator"`
	ValueText        *string    `db:"value_text"`
	ValueInteger     *int       `db:"value_integer"`
	ValueFloat       *float64   `db:"value_float"`
	ValueTimestamp   *time.Time `db:"value_timestamp"`
	ValueTimestamptz *time.Time `db:"value_timestamptz"`
}

type Processor interface {
	Name() string
	Process(ctx context.Context, content []byte) ([]BlobProperties, error)
}
