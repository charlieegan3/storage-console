package migration

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrations embed.FS

func Up(db *sql.DB, cfg *postgres.Config) error {
	m, err := buildMigrationsDriver(db, cfg)
	if err != nil {
		return fmt.Errorf("failed to build database driver to run up migrations: %w", err)
	}

	err = m.Up()
	if err != nil && err.Error() != migrate.ErrNoChange.Error() {
		return fmt.Errorf("failed to run database up migrations: %w", err)
	}

	return nil
}

func buildMigrationsDriver(db *sql.DB, cfg *postgres.Config) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating database driver: %w", err)
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("error loading migrations source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("error loading migrations instance: %w", err)
	}

	return m, nil
}

func Down(db *sql.DB, cfg *postgres.Config) error {
	m, err := buildMigrationsDriver(db, cfg)
	if err != nil {
		return fmt.Errorf("failed to build database driver to run down migrations: %w", err)
	}

	err = m.Down()
	if err != nil && err.Error() != migrate.ErrNoChange.Error() {
		return fmt.Errorf("failed to run database down migrations: %w", err)
	}

	return nil
}

func Cycle(db *sql.DB, cfg *postgres.Config) error {
	err := Down(db, cfg)
	if err != nil {
		return fmt.Errorf("failed to run down migrations: %w", err)
	}

	err = Up(db, cfg)
	if err != nil {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}
