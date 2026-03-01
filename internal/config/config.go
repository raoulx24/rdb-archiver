package config

import (
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

type Config struct {
	Source       snapshotwatcher.Config `yaml:"source"`
	Destination  worker.Config          `yaml:"destination"`
	WatchFS      watchfs.Config         `yaml:"watchFS"`
	Logging      logging.Config         `yaml:"logging"`
	ConfigReload ReloadConfig           `yaml:"configReload"`
}

type ReloadConfig struct {
	Enabled bool   `yaml:"enabled"`
	Method  string `yaml:"method"` // "fsnotify"
}
