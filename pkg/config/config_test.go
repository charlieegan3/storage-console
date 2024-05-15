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
database:
  connection_string: postgresql://postgres:password@localhost:5432
  migrations_table: storage_console
  params:
    dbname: storage_console
    sslmode: disable
buckets:
  local:
    url: http://127.0.0.1:9000
    access_key: minioadmin
    secret_key: minioadmin
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

	if config.Database.ConnectionString != "postgresql://postgres:password@localhost:5432?dbname=storage_console&sslmode=disable" {
		t.Fatalf("unexpected database connection string: %s", config.Database.ConnectionString)
	}

	if config.Database.MigrationsTable != "storage_console" {
		t.Fatalf("unexpected database migrations table: %s", config.Database.MigrationsTable)
	}

	if config.Buckets["local"].URL != "http://127.0.0.1:9000" {
		t.Fatalf("unexpected bucket url: %s", config.Buckets["local"].URL)
	}

	if config.Buckets["local"].AccessKey != "minioadmin" {
		t.Fatalf("unexpected bucket access key: %s", config.Buckets["local"].AccessKey)
	}

	if config.Buckets["local"].SecretKey != "minioadmin" {
		t.Fatalf("unexpected bucket secret key: %s", config.Buckets["local"].SecretKey)
	}
}
