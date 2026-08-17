package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenDatabase(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	ctx := t.Context()
	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})
}

func TestOpenDatabaseOnInvalidPath(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "missing", "test.db")
	ctx := t.Context()

	db, err := openDatabase(ctx, databasePath)
	if err == nil {
		_ = db.Close()
		t.Fatalf("openDatabase() error = nil, expected error")
	}

	if !strings.Contains(err.Error(), databasePath) {
		t.Errorf("openDatabase() error = %q, expected it to contain %q", err, databasePath)
	}
}

func TestOpenDatabasePragmas(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	ctx := t.Context()

	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	var foreignKeys int
	err = db.QueryRowContext(t.Context(), "PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}

	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, expected 1", foreignKeys)
	}

	var busyTimeout int
	err = db.QueryRowContext(t.Context(), "PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}

	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, expected 5000", busyTimeout)
	}

	var journalMode string
	err = db.QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, expected 'wal'", journalMode)
	}

	var synchronous int
	err = db.QueryRowContext(t.Context(), "PRAGMA synchronous").Scan(&synchronous)
	if err != nil {
		t.Fatalf("query synchronous: %v", err)
	}

	if synchronous != 2 {
		t.Errorf("synchronous = %d, expected 2", synchronous)
	}
}
