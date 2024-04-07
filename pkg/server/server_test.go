package server

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/charlieegan3/storage-console/pkg/config"
)

func TestNewServer(t *testing.T) {
	var err error
	ctx := context.Background()

	port, err := freePort()
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

}

func freePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}
