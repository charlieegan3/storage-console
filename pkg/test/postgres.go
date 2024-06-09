package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/charlieegan3/storage-console/pkg/database/migration"
	"github.com/charlieegan3/storage-console/pkg/utils"
)

func InitPostgres(ctx context.Context, t *testing.T) (db *sql.DB, postgresCleanup func() error, err error) {
	dbName := "storage_console"
	dbUser := "user"
	dbPassword := "password"

	databasePort, err := utils.FreePort(5433)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find free port for database: %s", err)
	}

	req := testcontainers.ContainerRequest{
		Image: "postgres:16.2",
		ExposedPorts: []string{
			fmt.Sprintf("%d:5432", databasePort),
		},
		Env: map[string]string{
			"POSTGRES_DB":       dbName,
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPassword,
		},
		WaitingFor: wait.ForLog("PostgreSQL init process complete"),
	}
	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not start postgres: %s", err)
	}

	connectionString := fmt.Sprintf(
		"postgresql://%s:%s@127.0.0.1:%d/%s?sslmode=disable",
		dbUser,
		dbPassword,
		databasePort,
		dbName,
	)

	t.Log("Postgres:", connectionString)

	db, err = sql.Open("postgres", connectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("could not open database connection: %s", err)
	}

	retries := 5
	for {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
		retries--
		if retries == 0 {
			return nil, nil, fmt.Errorf("could not connect to database before timeout: %s", err)
		}
	}

	err = migration.Cycle(db, &postgres.Config{
		MigrationsTable: "schema_migrations_storage_console",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not run migrations up and down: %s", err)
	}

	return db, func() error {
		if err := postgresContainer.Terminate(ctx); err != nil {
			return fmt.Errorf("could not terminate postgres: %s", err)
		}
		return nil
	}, nil
}
