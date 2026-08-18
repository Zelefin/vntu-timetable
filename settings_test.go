package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func openSettingsTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

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

	return db
}

func TestLoadWeekOffset(t *testing.T) {
	db := openSettingsTestDatabase(t)

	got, err := loadWeekOffset(t.Context(), db)
	if err != nil {
		t.Fatalf("loadWeekOffset() error = %v", err)
	}

	if got != 0 {
		t.Errorf("loadWeekOffset() = %d, expected 0", got)
	}
}

func TestLoadWeekOffsetReadsUpdatedValue(t *testing.T) {
	db := openSettingsTestDatabase(t)

	_, err := db.ExecContext(
		t.Context(),
		`UPDATE settings SET value = '1' WHERE key = 'week_offset'`,
	)
	if err != nil {
		t.Fatalf("update week_offset: %v", err)
	}

	got, err := loadWeekOffset(t.Context(), db)
	if err != nil {
		t.Fatalf("loadWeekOffset() error = %v", err)
	}

	if got != 1 {
		t.Errorf("loadWeekOffset() = %d, expected 1", got)
	}
}

func TestLoadWeekOffsetRejectsInvalidValue(t *testing.T) {
	db := openSettingsTestDatabase(t)

	_, err := db.ExecContext(
		t.Context(),
		`UPDATE settings SET value = '2' WHERE key = 'week_offset'`,
	)
	if err != nil {
		t.Fatalf("update week_offset: %v", err)
	}

	_, err = loadWeekOffset(t.Context(), db)
	if err == nil {
		t.Fatal("loadWeekOffset() error = nil, expected error")
	}
}

func TestLoadWeekOffsetRequiresSetting(t *testing.T) {
	db := openSettingsTestDatabase(t)

	_, err := db.ExecContext(
		t.Context(),
		`DELETE FROM settings WHERE key = 'week_offset'`,
	)
	if err != nil {
		t.Fatalf("delete week_offset: %v", err)
	}

	_, err = loadWeekOffset(t.Context(), db)
	if err == nil {
		t.Fatal("loadWeekOffset() error = nil, expected error")
	}

	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf(
			"loadWeekOffset() error = %v, expected it to wrap sql.ErrNoRows",
			err,
		)
	}
}

func TestUpdateWeekOffset(t *testing.T) {
	db := openSettingsTestDatabase(t)

	updatedAt := time.Date(
		2026,
		time.September,
		1,
		7,
		15,
		30,
		123456789,
		time.FixedZone("UTC+3", 3*60*60),
	)

	err := updateWeekOffset(t.Context(), db, 1, updatedAt)
	if err != nil {
		t.Fatalf("updateWeekOffset() error = %v", err)
	}

	var storedValue string
	var storedUpdatedAt string

	err = db.QueryRowContext(
		t.Context(),
		`
			SELECT value, updated_at
			FROM settings
			WHERE key = 'week_offset'
		`,
	).Scan(&storedValue, &storedUpdatedAt)
	if err != nil {
		t.Fatalf("read week_offset: %v", err)
	}

	if storedValue != "1" {
		t.Errorf("week_offset value = %q, expected %q", storedValue, "1")
	}

	expectedUpdatedAt := updatedAt.UTC().Format(time.RFC3339Nano)
	if storedUpdatedAt != expectedUpdatedAt {
		t.Errorf(
			"week_offset updated_at = %q, expected %q",
			storedUpdatedAt,
			expectedUpdatedAt,
		)
	}
}

func TestUpdateWeekOffsetRejectsInvalidInput(t *testing.T) {
	validTime := time.Date(
		2026,
		time.September,
		1,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name      string
		offset    int
		updatedAt time.Time
	}{
		{
			name:      "negative offset",
			offset:    -1,
			updatedAt: validTime,
		},
		{
			name:      "offset greater than one",
			offset:    2,
			updatedAt: validTime,
		},
		{
			name:      "zero update time",
			offset:    1,
			updatedAt: time.Time{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openSettingsTestDatabase(t)

			err := updateWeekOffset(
				t.Context(),
				db,
				test.offset,
				test.updatedAt,
			)
			if err == nil {
				t.Fatal("updateWeekOffset() error = nil, expected error")
			}

			got, err := loadWeekOffset(t.Context(), db)
			if err != nil {
				t.Fatalf("loadWeekOffset() error = %v", err)
			}

			if got != 0 {
				t.Errorf(
					"week offset after rejected update = %d, expected 0",
					got,
				)
			}
		})
	}
}

func TestUpdateWeekOffsetRequiresSetting(t *testing.T) {
	db := openSettingsTestDatabase(t)

	_, err := db.ExecContext(
		t.Context(),
		`DELETE FROM settings WHERE key = 'week_offset'`,
	)
	if err != nil {
		t.Fatalf("delete week_offset: %v", err)
	}

	updatedAt := time.Date(
		2026,
		time.September,
		1,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	err = updateWeekOffset(t.Context(), db, 1, updatedAt)
	if err == nil {
		t.Fatal("updateWeekOffset() error = nil, expected error")
	}
}
