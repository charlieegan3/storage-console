package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {

	rawConfig := strings.NewReader(`
server:
  port: 8080
  address: localhost
  log:
    error: stderr
    info: stdout
`)

	config, err := LoadConfig(rawConfig)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if config.Server.Port != 8080 {
		t.Fatalf("unexpected server port: %d", config.Server.Port)
	}

	if config.Server.Address != "localhost" {
		t.Fatalf("unexpected server address: %s", config.Server.Address)
	}

	if config.Server.LoggerError == nil {
		t.Fatalf("logger error was nil")
	}

	if config.Server.LoggerError.Writer() != os.Stderr {
		t.Fatalf("unexpected server logger error: %v", config.Server.LoggerError)
	}
}
