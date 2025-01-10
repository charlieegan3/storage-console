package runner

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/database"
	"github.com/charlieegan3/storage-console/pkg/properties"
	"github.com/charlieegan3/storage-console/pkg/properties/color"
	"github.com/charlieegan3/storage-console/pkg/properties/exif"
)

//go:embed needs_props.sql
var needsPropsSQL string

type Report struct {
	Counts map[string]int
}

type Options struct {
	SchemaName string
	BucketName string

	EnabledProcessors []string

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}

type blobProperties struct {
	ID       int
	MD5      string
	SetExif  bool
	SetColor bool
}

func Run(
	ctx context.Context,
	db *sql.DB,
	minioClient *minio.Client,
	opts *Options,
) (*Report, error) {
	processors := make(map[string]properties.Processor)
	for _, processorName := range opts.EnabledProcessors {
		processor, err := processorForName(processorName)
		if err != nil {
			return nil, fmt.Errorf("could not get processor: %s", err)
		}

		processors[processorName] = processor
	}

	txn, err := database.NewTxnWithSchema(db, opts.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %s", err)
	}

	var rpt Report
	rpt.Counts = make(map[string]int)

	rows, err := txn.QueryContext(ctx, needsPropsSQL)
	if err != nil {
		_ = txn.Rollback()
		return nil, fmt.Errorf("could not get blobs needing properties: %s", err)
	}

	var bps []blobProperties
	for rows.Next() {
		var bp blobProperties
		err = rows.Scan(&bp.ID, &bp.MD5, &bp.SetExif, &bp.SetColor)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not scan path: %s", err)
		}

		bps = append(bps, bp)
	}

	for _, bp := range bps {
		var props []properties.BlobProperties

		processorsNeeded := []string{}
		if !bp.SetExif {
			processorsNeeded = append(processorsNeeded, "exif")
		}
		if !bp.SetColor {
			processorsNeeded = append(processorsNeeded, "color")
		}

		for _, processorName := range processorsNeeded {
			ep, ok := processors[processorName]
			if !ok {
				return nil, fmt.Errorf("%s processor not found", processorName)
			}

			obj, err := minioClient.GetObject(
				ctx,
				opts.BucketName,
				path.Join("meta", processorName, bp.MD5+".json"),
				minio.GetObjectOptions{},
			)
			if err != nil {
				return nil, fmt.Errorf("could not get object: %s", err)
			}

			bs, err := io.ReadAll(obj)
			if err != nil {
				return nil, fmt.Errorf("could not read object: %s", err)
			}

			newProps, err := ep.Process(ctx, bs)
			if err != nil {
				return nil, fmt.Errorf("could not process object: %s", err)
			}

			props = append(props, properties.BlobProperties{
				PropertySource: processorName,
				PropertyType:   "Done",
				ValueType:      "Bool",
				ValueBool:      &[]bool{true}[0],
			})

			rpt.Counts[processorName] += len(newProps)

			props = append(props, newProps...)

			deleteOldPropsSQL := `
			delete from blob_properties
where source = $1 and blob_id = $2;`

			_, err = txn.Exec(deleteOldPropsSQL, processorName, bp.ID)
			if err != nil {
				return nil, fmt.Errorf("could not delete old properties: %s", err)
			}
		}

		err = insertBlobProperties(txn, bp.ID, props)
		if err != nil {
			return nil, fmt.Errorf("could not insert blob properties: %s", err)
		}
	}

	err = txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %s", err)
	}

	return &rpt, nil
}

func processorForName(name string) (properties.Processor, error) {
	switch name {
	case "exif":
		return &exif.ExifProcessor{}, nil
	case "color":
		return &color.ColorProcessor{}, nil
	}

	return nil, fmt.Errorf("unknown processor: %s", name)
}

func insertBlobProperties(tx *sql.Tx, blobID int, properties []properties.BlobProperties) error {
	if len(properties) == 0 {
		return nil
	}

	query := `
		INSERT INTO blob_properties (
			blob_id, source, property_type, value_type,
			value_bool, value_numerator, value_denominator,
			value_text, value_integer, value_float,
			value_timestamp, value_timestamptz
		) VALUES
	`

	values := []interface{}{}
	placeholders := []string{}

	for i, prop := range properties {
		start := i * 12 // 12 columns per row
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			start+1, start+2, start+3, start+4, start+5, start+6, start+7,
			start+8, start+9, start+10, start+11, start+12,
		))
		values = append(values,
			blobID,
			prop.PropertySource,
			prop.PropertyType,
			prop.ValueType,
			prop.ValueBool,
			prop.ValueNumerator,
			prop.ValueDenominator,
			prop.ValueText,
			prop.ValueInteger,
			prop.ValueFloat,
			prop.ValueTimestamp,
			prop.ValueTimestamptz,
		)
	}

	query += strings.Join(placeholders, ", ")

	_, err := tx.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to insert blob properties: %w", err)
	}

	return nil
}
