package meta

import (
	"context"

	"github.com/minio/minio-go/v7"
)

type ContentType int

const (
	JPG ContentType = iota
	JSON
)

func ContentTypeToString(contentType ContentType) string {
	switch contentType {
	case JPG:
		return "image/jpeg"
	case JSON:
		return "application/json"
	default:
		return ""
	}
}

func ContentTypeToFileExt(contentType ContentType) string {
	switch contentType {
	case JPG:
		return "jpg"
	case JSON:
		return "json"
	default:
		return ""
	}
}

type PutMetadata struct {
	Path        string
	ContentType ContentType
	Content     []byte
}

type MetadataOperationProcessor interface {
	Name() string
	ContentTypes() []string
	Process(
		ctx context.Context,
		objectInfo *minio.ObjectInfo,
		content []byte,
	) ([]PutMetadata, error)
}
