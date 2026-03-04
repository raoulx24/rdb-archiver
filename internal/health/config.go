package health

type Config struct {
	port uint16 `yaml:"port"`
}

func (c *Config) ApplyDefaults() {
	if c.port == 0 {
		c.port = 8080
	}
}
