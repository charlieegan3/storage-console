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
	"github.com/charlieegan3/storage-console/pkg/server/handlers"
)

func NewServer(db *sql.DB, minioClients map[string]*minio.Client, cfg *config.Config) (Server, error) {
	return Server{
		cfg:          cfg,
		db:           db,
		minioClients: minioClients,
	}, nil
}

type Server struct {
	cfg *config.Config

	db           *sql.DB
	minioClients map[string]*minio.Client

	httpServer *http.Server
}

func (s *Server) Start(ctx context.Context) error {
	var err error

	mux := http.NewServeMux()
	if s.cfg.Server.RegisterMux {
		mux, err = newMux(
			&handlers.Options{
				DevMode:                s.cfg.Server.DevMode,
				ObjectStorageProviders: s.minioClients,
				DB:                     s.db,
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

		var defaultBucket config.Bucket
		for _, bucket := range s.cfg.Buckets {
			if bucket.Default {
				defaultBucket = bucket
				break
			}
		}

		if defaultBucket.Provider == "" {
			log.Fatalf("no default bucket found")
		}

		mc, ok := s.minioClients[defaultBucket.Provider]
		if !ok {
			log.Fatalf("no minio client found for provider %s", defaultBucket.Provider)
		}

		_, err := importer.Run(ctx, s.db, mc, &importer.Options{
			StorageProviderName: "local",
			BucketName:          "local",
			SchemaName:          "storage_console",
		})
		if err != nil {
			log.Printf("error running importer: %v", err)
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
