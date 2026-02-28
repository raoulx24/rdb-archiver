package config

import "time"

type Config struct {
	Source       SourceConfig      `yaml:"source"`
	Destination  DestinationConfig `yaml:"destination"`
	Logging      LoggingConfig     `yaml:"logging"`
	ConfigReload ReloadConfig      `yaml:"configReload"`
}

type SourceConfig struct {
	Path        string      `yaml:"path"`
	PrimaryName string      `yaml:"primaryName"`
	AuxNames    []string    `yaml:"auxNames"`
	Watch       WatchConfig `yaml:"watch"`
}

type WatchConfig struct {
	Mode            string        `yaml:"mode"`           // "auto", "poll", "fsnotify"
	PollInterval    time.Duration `yaml:"pollInterval"`   // e.g. 5s
	DebounceWindow  time.Duration `yaml:"debounceWindow"` // e.g. 500ms
	StabilityWindow time.Duration `yaml:"stabilityWindow"`
}

type DestinationConfig struct {
	Root           string          `yaml:"root"`
	SubDir         string          `yaml:"subDir"`
	SnapshotSubdir string          `yaml:"snapshotSubdir"`
	Retention      RetentionConfig `yaml:"retention"`
}

type RetentionConfig struct {
	LastCount int             `yaml:"lastCount"`
	Rules     []RetentionRule `yaml:"rules"`
}

type RetentionRule struct {
	Name  string `yaml:"name"`
	Cron  string `yaml:"cron"`
	Count int    `yaml:"count"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`  // "info", "debug", etc.
	Format string `yaml:"format"` // "json", "text"
}

type ReloadConfig struct {
	Enabled bool   `yaml:"enabled"`
	Method  string `yaml:"method"` // "fsnotify"
}
