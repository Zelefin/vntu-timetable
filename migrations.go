package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int
	path    string
}

var availableMigrations = []migration{
	{version: 1, path: "migrations/001_initial.sql"},
}

func migrateDatabase(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY CHECK (version > 0),
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		) STRICT;
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, migration := range availableMigrations {
		var applied bool

		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM schema_migrations
				WHERE version=?
			)
		`, migration.version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", migration.version, err)
		}

		if applied {
			continue
		}

		migrationSQL, err := migrationFiles.ReadFile(migration.path)
		if err != nil {
			return fmt.Errorf("read migration %d from %q: %w", migration.version, migration.path, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", migration.version, err)
		}

		_, err = tx.ExecContext(ctx, string(migrationSQL))
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}

		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO schema_migrations (version) VALUES (?)",
			migration.version,
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", migration.version, err)
		}
	}

	return nil
}
