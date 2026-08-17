package main

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dbPath := "./abc"
	t.Setenv("DATABASE_PATH", dbPath)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.DatabasePath != dbPath {
		t.Errorf("cfg.DatabasePath = %q, expected %q", cfg.DatabasePath, dbPath)
	}
}

func TestLoadConfigRequiresDBPath(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	_, err := loadConfig()
	if err == nil {
		t.Fatalf("loadConfig() err = nil, expected error")
	}

	expErr := "DATABASE_PATH is required"
	if err.Error() != expErr {
		t.Errorf("loadConfig() error = %q, expected %q", err.Error(), expErr)
	}
}
