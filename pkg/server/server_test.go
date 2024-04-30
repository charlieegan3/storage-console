package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/charlieegan3/storage-console/pkg/config"
	"github.com/charlieegan3/storage-console/pkg/utils"
)

func TestNewServer(t *testing.T) {
	var err error
	ctx := context.Background()

	port, err := utils.FreePort()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	serverConfig := &config.Config{
		Server: config.Server{
			DevMode: true, // needed to pass auth middleware
			Port:    port,
			Address: "localhost",
		},
	}

	server, err := NewServer(serverConfig)
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

	if !strings.Contains(string(bodyBs), "Storage Console") {
		t.Fatalf("unexpected body: %s", bodyBs)
	}
}
