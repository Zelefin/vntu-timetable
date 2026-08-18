package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

func loadWeekOffset(ctx context.Context, db *sql.DB) (int, error) {
	var value string

	err := db.QueryRowContext(
		ctx,
		`SELECT value FROM settings WHERE key = 'week_offset'`,
	).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("load week offset: %w", err)
	}

	switch value {
	case "0":
		return 0, nil
	case "1":
		return 1, nil
	default:
		return 0, fmt.Errorf("load week offset: invalid value %q", value)
	}
}

func updateWeekOffset(
	ctx context.Context,
	db *sql.DB,
	offset int,
	updatedAt time.Time,
) error {
	if offset != 0 && offset != 1 {
		return fmt.Errorf("week offset must be 0 or 1, got %d", offset)
	}

	if updatedAt.IsZero() {
		return fmt.Errorf("week offset update time must not be zero")
	}

	result, err := db.ExecContext(
		ctx,
		`
			UPDATE settings
			SET value = ?, updated_at = ?
			WHERE key = 'week_offset'
		`,
		strconv.Itoa(offset),
		updatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("update week offset: %w", err)
	}

	updatedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated week offset row count: %w", err)
	}

	if updatedRows != 1 {
		return fmt.Errorf(
			"update week offset: updated %d rows, expected 1",
			updatedRows,
		)
	}

	return nil
}
