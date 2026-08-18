package main

import (
	"path/filepath"
	"testing"
)

func TestRunMigrate(t *testing.T) {
	ctx := t.Context()
	databasePath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("DATABASE_PATH", databasePath)

	if err := run(ctx, []string{"migrate"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

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
