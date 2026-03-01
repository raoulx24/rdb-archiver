package worker

import "github.com/raoulx24/rdb-archiver/internal/retention"

type Config struct {
	Root           string          `yaml:"root"`
	SubDir         string          `yaml:"subDir"`
	SnapshotSubdir string          `yaml:"snapshotSubdir"`
	Retention      RetentionConfig `yaml:"retention"`
}

type RetentionConfig struct {
	LastCount int                       `yaml:"lastCount"`
	Rules     []retention.RetentionRule `yaml:"rules"`
}
