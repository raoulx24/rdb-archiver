package worker

import "github.com/raoulx24/rdb-archiver/internal/retention"

type Config struct {
	Root           string          `yaml:"root"`
	SubDir         string          `yaml:"subDir"`
	SnapshotSubdir string          `yaml:"snapshotSubdir"`
	Retention      RetentionConfig `yaml:"retention"`
}

type RetentionConfig struct {
	LastCount            int              `yaml:"lastCount"`
	RemoveUnknownFolders bool             `yaml:"removeUnknownFolders"`
	Rules                []retention.Rule `yaml:"rules"`
}

func (c *Config) ApplyDefaults() {
	if c.Root == "" {
		c.Root = "/var/backups/redis"
	}
	if c.SubDir == "" {
		c.SubDir = "incoming"
	}
	if c.SnapshotSubdir == "" {
		c.SnapshotSubdir = "snapshots"
	}
	c.Retention.ApplyDefaults()
}

func (c *RetentionConfig) ApplyDefaults() {
	if c.LastCount == 0 {
		c.LastCount = 5 // keep last 5 snapshots
	}
	// Rules slice can stay empty; no default needed.
}
