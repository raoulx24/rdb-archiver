package prometheus

type Config struct {
	Enabled bool
	Port    uint16
}

func (c *Config) ApplyDefaults() {
	if c.Port == 0 {
		c.Port = 9090
	}
}
