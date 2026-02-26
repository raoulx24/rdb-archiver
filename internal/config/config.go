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
	Mode           string        `yaml:"mode"`            // "auto", "poll", "fsnotify"
	PollInterval   time.Duration `yaml:"poll_interval"`   // e.g. 5s
	DebounceWindow time.Duration `yaml:"debounce_window"` // e.g. 500ms
}

type DestinationConfig struct {
	Root           string          `yaml:"root"`
	SubDirEnv      string          `yaml:"subDirEnv"`
	SnapshotSubdir string          `yaml:"snapshot_subdir"`
	Retention      RetentionConfig `yaml:"retention"`
}

type RetentionConfig struct {
	LastCount int             `yaml:"last_count"`
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
