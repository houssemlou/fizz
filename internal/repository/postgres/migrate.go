package postgres

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrate runs all pending up migrations against the given database URL.
// It is idempotent: already-applied migrations are skipped.
func Migrate(databaseURL string, migrationsFS fs.FS) error {
	src, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5URL(databaseURL))
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// toPgx5URL converts a standard postgres:// URL to the pgx5:// scheme
// required by the golang-migrate pgx/v5 driver.
func toPgx5URL(dsn string) string {
	dsn = strings.Replace(dsn, "postgresql://", "pgx5://", 1)
	dsn = strings.Replace(dsn, "postgres://", "pgx5://", 1)
	return dsn
}
