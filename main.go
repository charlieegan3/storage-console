package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/charlieegan3/storage-console/pkg/config"
	"github.com/charlieegan3/storage-console/pkg/database/migration"
	"github.com/charlieegan3/storage-console/pkg/server"
)

func main() {
	ctx := context.Background()

	if len(os.Args) != 2 {
		log.Fatal("Please provide config as first arg")
	}

	configFile, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0644)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("error parsing config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.Database.ConnectionString)
	if err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}

	err = migration.Up(db, &postgres.Config{
		MigrationsTable: cfg.Database.MigrationsTable,
	})
	if err != nil {
		log.Fatalf("error running migrations: %v", err)
	}

	objectStorageProviders := make(map[string]*minio.Client)
	for k, p := range cfg.ObjectStorageProviders {
		minioClient, err := minio.New(p.URL, &minio.Options{
			Creds:  credentials.NewStaticV4(p.AccessKey, p.SecretKey, ""),
			Secure: false,
		})
		if err != nil {
			log.Fatalf("error connecting to minio: %v", err)
		}

		_, err = minioClient.ListBuckets(ctx)
		if err != nil {
			log.Fatalf("error listing buckets when testing minio connection: %v", err)
		}

		objectStorageProviders[k] = minioClient
	}

	srv, err := server.NewServer(db, objectStorageProviders, cfg)
	if err != nil {
		log.Fatalf("error creating server: %v", err)
	}

	if logger := cfg.Server.LoggerInfo; logger != nil {
		logger.Printf(
			"Starting server on http://%s:%d\n",
			cfg.Server.Address,
			cfg.Server.Port,
		)
	}

	err = srv.Start(ctx)
	if err != nil {
		log.Fatalf("error starting server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received %v, shutting down...\n", sig)

		err = srv.Stop(ctx)
		if err != nil {
			log.Fatalf("error stopping server: %v", err)
		}

		os.Exit(0)
	}()

	log.Println("Press Ctrl+C to exit.")

	select {}
}
