package meta

import (
	"github.com/minio/minio-go/v7"
)

type PutMetadata struct {
	Path    string
	Content []byte
	minio.PutObjectOptions
}

type MetadataOperationProcessor interface {
	Process(objectInfo minio.ObjectInfo, content []byte) ([]PutMetadata, error)
}
