// Package config defines the runtime configuration for rdb-archiver and helpers to load it.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values used by the service.
type Config struct {
	RDBDir            string
	ArchiveDir        string
	PollInterval      time.Duration
	RetentionKeepLast int
}

// Load reads configuration from environment variables with sensible defaults.
//
// RDB_ARCHIVER_RDB_DIR          (default: "./data")
// RDB_ARCHIVER_ARCHIVE_DIR      (default: "./archive")
// RDB_ARCHIVER_POLL_INTERVAL    (default: "2s")
// RDB_ARCHIVER_RETENTION_KEEP   (default: "10")
func Load() (Config, error) {
	cfg := Config{
		RDBDir:            getEnv("RDB_ARCHIVER_RDB_DIR", "./data"),
		ArchiveDir:        getEnv("RDB_ARCHIVER_ARCHIVE_DIR", "./archive"),
		RetentionKeepLast: mustInt(getEnv("RDB_ARCHIVER_RETENTION_KEEP", "10")),
	}

	pollStr := getEnv("RDB_ARCHIVER_POLL_INTERVAL", "2s")
	d, err := time.ParseDuration(pollStr)
	if err != nil {
		return Config{}, err
	}
	cfg.PollInterval = d

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
