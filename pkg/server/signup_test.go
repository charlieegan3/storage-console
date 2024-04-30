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

func TestSignupFlow(t *testing.T) {
	var err error
	ctx := context.Background()

	port, err := utils.FreePort()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	serverConfig := &config.Config{
		Server: config.Server{
			Port:    port,
			Address: "localhost",
		},
		WebAuthn: config.WebAuthn{
			Host:    "example.com",
			Origins: []string{"http://example.com", "http://foo.example.com"},
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

	// 1. test the index page links
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

	if !strings.Contains(string(bodyBs), "Register") {
		t.Fatalf("unexpected body: %s", bodyBs)
	}

	// 2. test the register page
	req, err = http.NewRequest(
		"GET",
		fmt.Sprintf("http://localhost:%d/register", port),
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	bodyBs, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Logf("body: %s", bodyBs)
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	if !strings.Contains(string(bodyBs), "Username") {
		t.Fatalf("unexpected body: %s", bodyBs)
	}

	// 3. test the register page with a post
	req, err = http.NewRequest(
		"GET",
		fmt.Sprintf("http://localhost:%d/register/begin/exampleuser", port),
		nil,
	)
	req.Header.Add("Accept", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	bodyBs, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Logf("body: %s", bodyBs)
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

}
