package main

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	DatabasePath string
}

func loadConfig() (Config, error) {
	databasePath, ok := os.LookupEnv("DATABASE_PATH")
	if !ok || strings.TrimSpace(databasePath) == "" {
		return Config{}, errors.New("DATABASE_PATH is required")
	}

	return Config{
		DatabasePath: databasePath,
	}, nil
}
