package watchfs

import "time"

type Config struct {
	FSNotify        FSNotifyConfig `yaml:"fsnotify"`
	Pool            PoolConfig     `yaml:"pool"`
	StabilityWindow string         `yaml:"stabilityWindow"`
}

type FSNotifyConfig struct {
	DebounceWindow string `yaml:"debounceWindow"`
}

type PoolConfig struct {
	Interval string `yaml:"interval"`
}

func (c *Config) ApplyDefaults() {
	if c.FSNotify.DebounceWindow == "" || !isValidDuration(c.FSNotify.DebounceWindow) {
		c.FSNotify.DebounceWindow = "150ms"
	}

	if c.Pool.Interval == "" || !isValidDuration(c.Pool.Interval) {
		c.Pool.Interval = "5s"
	}

	if c.StabilityWindow == "" || !isValidDuration(c.StabilityWindow) {
		c.StabilityWindow = "200ms"
	}
}

func isValidDuration(s string) bool {
	_, err := time.ParseDuration(s)
	return err == nil
}
