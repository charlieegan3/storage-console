package server

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/charlieegan3/storage-console/pkg/config"
	"github.com/charlieegan3/storage-console/pkg/utils"
)

func TestNewServer(t *testing.T) {
	var err error
	ctx := context.Background()

	port, err := utils.FreePort(3000)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	serverConfig := &config.Config{
		Server: config.Server{
			DevMode:     true, // needed to pass auth middleware
			Port:        port,
			Address:     "localhost",
			RegisterMux: false,
			RunImporter: false,
		},
		S3: config.S3{
			Endpoint:   "localhost:9000",
			AccessKey:  "minio",
			SecretKey:  "minio123",
			BucketName: "local",
		},
	}

	var db *sql.DB

	minioClient, err := minio.New(serverConfig.S3.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			serverConfig.S3.AccessKey,
			serverConfig.S3.SecretKey,
			"",
		),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("error connecting to minio: %v", err)
	}

	server, err := NewServer(db, minioClient, serverConfig)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer func() {
		err := server.Stop(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}()

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if conn != nil {
		err := conn.Close()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://localhost:%d/", port),
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	bodyBs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Logf("body: %s", bodyBs)
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}
