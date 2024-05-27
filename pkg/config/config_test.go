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
  dev_mode: true
  log:
    error: stderr
    info: stdout
  register_mux: true
  run_importer: true
database:
  connection_string: postgresql://postgres:password@localhost:5432
  migrations_table: schema_migrations_storage_console
  params:
    dbname: storage_console
    sslmode: disable
object_storage_providers:
  local-minio:
    url: "127.0.0.1:9000"
    access_key: minioadmin
    secret_key: minioadmin
buckets:
  local:
    provider: local-minio
    default: true
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

	if config.Server.RegisterMux != true {
		t.Fatalf("unexpected server register mux: %v", config.Server.RegisterMux)
	}

	if config.Server.RunImporter != true {
		t.Fatalf("unexpected server run importer: %v", config.Server.RunImporter)
	}

	if config.Database.ConnectionString != "postgresql://postgres:password@localhost:5432?dbname=storage_console&sslmode=disable" {
		t.Fatalf("unexpected database connection string: %s", config.Database.ConnectionString)
	}

	if config.Database.MigrationsTable != "schema_migrations_storage_console" {
		t.Fatalf("unexpected database migrations table: %s", config.Database.MigrationsTable)
	}

	if config.Buckets["local"].Provider != "local-minio" {
		t.Fatalf("unexpected bucket provider: %s", config.Buckets["local"].Provider)
	}
	if config.ObjectStorageProviders["local-minio"].URL != "127.0.0.1:9000" {
		t.Fatalf("unexpected bucket url: %s", config.ObjectStorageProviders["local-minio"].URL)
	}

	if config.ObjectStorageProviders["local-minio"].AccessKey != "minioadmin" {
		t.Fatalf("unexpected bucket access key: %s", config.ObjectStorageProviders["local-minio"].AccessKey)
	}

	if config.ObjectStorageProviders["local-minio"].SecretKey != "minioadmin" {
		t.Fatalf("unexpected bucket secret key: %s", config.ObjectStorageProviders["local-minio"].SecretKey)
	}
}
