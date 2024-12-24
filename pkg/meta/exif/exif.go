package exif

import (
	"encoding/json"
	"fmt"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/meta"
)

type ExifMetadataProcessor struct{}

func (p *ExifMetadataProcessor) Name() string {
	return "exif"
}

func (p *ExifMetadataProcessor) Process(objectInfo *minio.ObjectInfo, content []byte) ([]meta.PutMetadata, error) {
	metadata := make(map[string]interface{})

	rawExif, err := exif.SearchAndExtractExif(content)
	if err == exif.ErrNoExif {
		return []meta.PutMetadata{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get raw exif data: %w", err)
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, fmt.Errorf("failed to create IFD mapping: %w", err)
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return nil, fmt.Errorf("failed to collect exif data: %w", err)
	}

	cb := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		tagName := ite.TagName()
		rawValue, err := ite.Value()
		if err != nil {
			return fmt.Errorf("could not get value for tag %s: %w", tagName, err)
		}

		metadata[tagName] = rawValue
		return nil
	}

	err = index.RootIfd.EnumerateTagsRecursively(cb)
	if err != nil {
		return nil, fmt.Errorf("failed to walk exif data tree: %w", err)
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("error converting EXIF data to JSON: %w", err)
	}

	putMetadata := meta.PutMetadata{
		ContentType: meta.JSON,
		Content:     jsonData,
	}

	return []meta.PutMetadata{putMetadata}, nil
}
