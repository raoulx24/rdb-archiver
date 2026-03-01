package snapshotwatcher

type Config struct {
	Path        string   `yaml:"path"`
	PrimaryName string   `yaml:"primaryName"`
	AuxNames    []string `yaml:"auxNames"`
	WatchMode   string   `yaml:"watchMode"`
}

func (c *Config) ApplyDefaults() {
	if c.Path == "" {
		c.Path = "/data"
	}
	if c.PrimaryName == "" {
		c.PrimaryName = "dump.rdb"
	}
	if c.WatchMode == "" {
		c.WatchMode = "fsnotify" // "auto" | "fsnotify" | "poll"
	}
	// AuxNames can stay empty; no default needed.
}
