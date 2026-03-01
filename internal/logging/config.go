package logging

type Config struct {
	Level  string `yaml:"level"`  // "info", "debug", etc.
	Format string `yaml:"format"` // "json", "text"
}

func (c *Config) ApplyDefaults() {
	if c.Level == "" {
		c.Level = "info"
	}
	if c.Format == "" {
		c.Format = "text" // or "json"
	}
}
