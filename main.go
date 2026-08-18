package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: vntu-timetable migrate")
	}

	switch args[0] {
	case "migrate":
		if len(args) > 1 {
			return errors.New("migrate does not accept arguments")
		}

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}

		db, err := openDatabase(ctx, cfg.DatabasePath)
		if err != nil {
			return err
		}

		if err := migrateDatabase(ctx, db); err != nil {
			_ = db.Close()
			return fmt.Errorf("migrate database: %w", err)
		}

		if err := db.Close(); err != nil {
			return fmt.Errorf("close database: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
