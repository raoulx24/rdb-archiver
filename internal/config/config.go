package config

import "time"

// Holds all runtime configuration for the service

type Config struct {
	RDBDir       string
	ArchiveDir   string
	PollInterval time.Duration
}
