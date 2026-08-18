package main

import (
	"path/filepath"
	"testing"
)

func TestMigrateDatabaseAppliesMigrations(t *testing.T) {
	ctx := t.Context()
	databasePath := filepath.Join(t.TempDir(), "test.db")

	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	if err := migrateDatabase(ctx, db); err != nil {
		t.Fatalf("migrateDatabase() error = %v", err)
	}

	expectedTables := []string{
		"schema_migrations",
		"faculties",
		"groups",
		"lessons",
		"users",
		"settings",
	}

	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM sqlite_schema
				WHERE type = 'table' AND name = ?
			)
		`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %q: %v", table, err)
		}

		if !exists {
			t.Errorf("table %q does not exist", table)
		}
	}

	for _, expectedMigration := range availableMigrations {
		var versionRecorded bool
		err = db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM schema_migrations
				WHERE version = ?
			)
		`, expectedMigration.version).Scan(&versionRecorded)
		if err != nil {
			t.Fatalf("check migration version %d: %v", expectedMigration.version, err)
		}

		if !versionRecorded {
			t.Errorf("migration version %d was not recorded", expectedMigration.version)
		}
	}

	var weekOffset string
	err = db.QueryRowContext(ctx, `
		SELECT value
		FROM settings
		WHERE key = 'week_offset'
	`).Scan(&weekOffset)
	if err != nil {
		t.Fatalf("read week_offset: %v", err)
	}

	if weekOffset != "0" {
		t.Errorf("week_offset = %q, expected %q", weekOffset, "0")
	}
}

func TestMigrateDatabaseSkipsAppliedMigrations(t *testing.T) {
	ctx := t.Context()
	databasePath := filepath.Join(t.TempDir(), "test.db")

	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	if err := migrateDatabase(ctx, db); err != nil {
		t.Fatalf("first migrateDatabase() error = %v", err)
	}

	if err := migrateDatabase(ctx, db); err != nil {
		t.Fatalf("second migrateDatabase() error = %v", err)
	}

	var appliedCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM schema_migrations
	`).Scan(&appliedCount)
	if err != nil {
		t.Fatalf("count migration records: %v", err)
	}

	if appliedCount != len(availableMigrations) {
		t.Errorf(
			"migration record count = %d, expected %d",
			appliedCount,
			len(availableMigrations),
		)
	}
}

func TestMigrateDatabaseRollsBackFailedMigration(t *testing.T) {
	ctx := t.Context()
	databasePath := filepath.Join(t.TempDir(), "test.db")

	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	// Migration 1 creates faculties first and groups second.
	// This existing table makes it fail after faculties has been created.
	_, err = db.ExecContext(ctx, `
		CREATE TABLE groups (
			id INTEGER PRIMARY KEY
		) STRICT
	`)
	if err != nil {
		t.Fatalf("create conflicting groups table: %v", err)
	}

	err = migrateDatabase(ctx, db)
	if err == nil {
		t.Fatal("migrateDatabase() error = nil, expected error")
	}

	var facultiesExist bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sqlite_schema
			WHERE type = 'table' AND name = 'faculties'
		)
	`).Scan(&facultiesExist)
	if err != nil {
		t.Fatalf("check faculties table: %v", err)
	}

	if facultiesExist {
		t.Error("faculties table exists after failed migration")
	}

	var appliedCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM schema_migrations
		WHERE version = 1
	`).Scan(&appliedCount)
	if err != nil {
		t.Fatalf("count migration records: %v", err)
	}

	if appliedCount != 0 {
		t.Errorf("migration version 1 record count = %d, expected 0", appliedCount)
	}
}
