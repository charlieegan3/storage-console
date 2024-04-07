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
webauthn:
  host: example.com
  origins:
  - http://example.com
  - http://foo.example.com
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

	if config.WebAuthn.Host != "example.com" {
		t.Errorf("unexpected webauthn host: %s", config.WebAuthn.Host)
	}

	if len(config.WebAuthn.Origins) != 2 {
		t.Errorf("unexpected number of webauthn origins: %d", len(config.WebAuthn.Origins))
	}

	if config.WebAuthn.Origins[0] != "http://example.com" {
		t.Errorf("unexpected webauthn origin: %s", config.WebAuthn.Origins[0])
	}

	if config.WebAuthn.Origins[1] != "http://foo.example.com" {
		t.Errorf("unexpected webauthn origin: %s", config.WebAuthn.Origins[1])
	}
}
