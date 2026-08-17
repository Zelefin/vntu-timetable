package main

import (
	"path/filepath"
	"testing"
)

func TestMigrateDatabaseAppliesInitialMigration(t *testing.T) {
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

	var versionRecorded bool
	err = db.QueryRowContext(ctx, `
	SELECT EXISTS (
		SELECT 1
		FROM schema_migrations
		WHERE version = 1
	)
`).Scan(&versionRecorded)
	if err != nil {
		t.Fatalf("check migration version: %v", err)
	}

	if !versionRecorded {
		t.Error("migration version 1 was not recorded")
	}
}

func TestMigrateDatabaseSkipsAppliedMigration(t *testing.T) {
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
  		WHERE version = 1
  	`).Scan(&appliedCount)
	if err != nil {
		t.Fatalf("count migration records: %v", err)
	}

	if appliedCount != 1 {
		t.Errorf("migration version 1 record count = %d, expected 1", appliedCount)
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
