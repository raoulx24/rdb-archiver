package fs

import "time"

type Config struct {
	MaxRetries       int    `yaml:"maxRetries"`
	RetryBase        string `yaml:"retryBase"`
	RetryDurationCap string `yaml:"retryDurationCap"`
}

func (c *Config) ApplyDefaults() {
	if c.MaxRetries <= 0 {
		c.MaxRetries = 7
	}
	if c.RetryBase == "" || !isValidDuration(c.RetryBase) {
		c.RetryBase = "50ms"
	}
	if c.RetryDurationCap == "" || !isValidDuration(c.RetryDurationCap) {
		c.RetryDurationCap = "1s"
	}
}

func isValidDuration(s string) bool {
	_, err := time.ParseDuration(s)
	return err == nil
}
