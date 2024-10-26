package handlers

import (
	"database/sql"
	"log"

	"github.com/minio/minio-go/v7"
)

type Options struct {
	DevMode    bool
	EtagScript string
	EtagStyles string

	LoggerError *log.Logger
	LoggerInfo  *log.Logger

	DB            *sql.DB
	ObjectStorage *minio.Client
	BucketName    string
}
