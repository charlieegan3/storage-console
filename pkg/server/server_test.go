package server

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/minio/minio-go/v7"

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
		ObjectStorageProviders: map[string]config.ObjectStorageProvider{
			"local-minio": {
				URL:       "http://localhost:9000",
				AccessKey: "minio",
				SecretKey: "minio123",
			},
		},
		Buckets: map[string]config.Bucket{
			"local": {
				Provider: "local-minio",
				Default:  true,
			},
		},
	}

	var db *sql.DB
	minioClients := map[string]*minio.Client{
		"local-minio": nil,
	}

	server, err := NewServer(db, minioClients, serverConfig)
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

	if resp.StatusCode != http.StatusNotFound {
		t.Logf("body: %s", bodyBs)
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}
