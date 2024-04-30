package config

import (
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {

	rawConfig := strings.NewReader(`
server:
  port: 8080
  address: localhost
`)

	config, err := LoadConfig(rawConfig)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if config.Server.Port != 8080 {
		t.Errorf("unexpected server port: %d", config.Server.Port)
	}

	if config.Server.Address != "localhost" {
		t.Errorf("unexpected server address: %s", config.Server.Address)
	}
}
