package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/config"
	"github.com/charlieegan3/storage-console/pkg/importer"
	"github.com/charlieegan3/storage-console/pkg/meta/thumbnail"
	"github.com/charlieegan3/storage-console/pkg/server/handlers"
)

func NewServer(db *sql.DB, minioClient *minio.Client, cfg *config.Config) (Server, error) {
	return Server{
		cfg:         cfg,
		db:          db,
		minioClient: minioClient,
	}, nil
}

type Server struct {
	cfg *config.Config

	db          *sql.DB
	minioClient *minio.Client

	httpServer *http.Server
}

func (s *Server) Start(ctx context.Context) error {
	var err error

	mux := http.NewServeMux()
	if s.cfg.Server.RegisterMux {
		mux, err = newMux(
			&handlers.Options{
				DevMode:    s.cfg.Server.DevMode,
				DB:         s.db,
				S3:         s.minioClient,
				BucketName: s.cfg.S3.BucketName,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create mux: %w", err)
		}
	}

	s.httpServer = &http.Server{
		Addr: fmt.Sprintf(
			"%s:%d",
			s.cfg.Server.Address,
			s.cfg.Server.Port,
		),
		Handler: mux,
	}

	go func() {
		if !s.cfg.Server.RunImporter {
			return
		}

		_, err := importer.Run(ctx, s.db, s.minioClient, &importer.Options{
			BucketName:  s.cfg.S3.BucketName,
			SchemaName:  "storage_console",
			LoggerInfo:  s.cfg.Server.LoggerInfo,
			LoggerError: s.cfg.Server.LoggerError,
		})
		if err != nil {
			log.Printf("error running importer: %v", err)
			return
		}

		_, err = thumbnail.Run(ctx, s.db, s.minioClient, &thumbnail.Options{
			SchemaName:   "storage_console",
			BucketName:   s.cfg.S3.BucketName,
			ThumbMaxSize: 300,
			LoggerInfo:   s.cfg.Server.LoggerInfo,
			LoggerError:  s.cfg.Server.LoggerError,
		})
		if err != nil {
			log.Printf("error running thumbnail: %v", err)
			return
		}
		log.Println("imported")
	}()

	go func() {
		<-ctx.Done()
		err = s.httpServer.Shutdown(ctx)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		err = s.httpServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			return err
		}
	}

	s.httpServer = nil

	return nil
}
