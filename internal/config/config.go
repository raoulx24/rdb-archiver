// Package config defines the full configuration model for rdb-archiver,
// including defaults, YAML loading, and environment overrides.
package config

import (
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the entire configuration file.
type Config struct {
	Source struct {
		Path              string `yaml:"path"`
		RDBFilename       string `yaml:"rdb_filename"`
		NodesConfFilename string `yaml:"nodes_conf_filename"`
	} `yaml:"source"`

	Destination struct {
		Root           string `yaml:"root"`
		SnapshotSubdir string `yaml:"snapshot_subdir"`
	} `yaml:"destination"`

	Hostname struct {
		OverrideEnv string `yaml:"override_env"`
		Fallback    string `yaml:"fallback"`
	} `yaml:"hostname"`

	Watch struct {
		Mode           string        `yaml:"mode"`
		PollInterval   time.Duration `yaml:"poll_interval"`
		DebounceWindow time.Duration `yaml:"debounce_window"`
	} `yaml:"watch"`

	Retention struct {
		LastCount int             `yaml:"last_count"`
		Rules     []RetentionRule `yaml:"rules"`
	} `yaml:"retention"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`

	Reload struct {
		Enabled bool   `yaml:"enabled"`
		Method  string `yaml:"method"`
	} `yaml:"reload"`
}

// RetentionRule defines a single cron-based retention tier.
type RetentionRule struct {
	Name  string `yaml:"name"`
	Cron  string `yaml:"cron"`
	Count int    `yaml:"count"`
}

func defaultConfig() Config {
	var c Config

	c.Source.Path = "/data"
	c.Source.RDBFilename = "dump.rdb"
	c.Source.NodesConfFilename = "nodes.conf"

	c.Destination.Root = "/dest"
	c.Destination.SnapshotSubdir = "last"

	c.Hostname.OverrideEnv = "HOSTNAME_OVERRIDE"
	c.Hostname.Fallback = "auto"

	c.Watch.Mode = "auto"
	c.Watch.PollInterval = 5 * time.Second
	c.Watch.DebounceWindow = 500 * time.Millisecond

	c.Retention.LastCount = 5

	c.Logging.Level = "info"
	c.Logging.Format = "json"

	c.Reload.Enabled = true
	c.Reload.Method = "fsnotify"

	return c
}

func Load(path string) (Config, error) {
	cfg := defaultConfig()

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	applyEnvOverrides(&cfg)

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	// Override hostname via environment
	if v := os.Getenv(cfg.Hostname.OverrideEnv); v != "" {
		cfg.Hostname.Fallback = v
	}

	// Override logging level
	if v := os.Getenv("RDB_ARCHIVER_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}

	// Override retention last_count
	if v := os.Getenv("RDB_ARCHIVER_RETENTION_LAST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Retention.LastCount = n
		}
	}
}
