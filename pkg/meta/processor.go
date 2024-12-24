package meta

import (
	"github.com/minio/minio-go/v7"
)

type ContentType int

const (
	JPG ContentType = iota
	JSON
)

type PutMetadata struct {
	ContentType ContentType
	Content     []byte
}

type MetadataOperationProcessor interface {
	Name() string
	Process(objectInfo *minio.ObjectInfo, content []byte) ([]PutMetadata, error)
}
