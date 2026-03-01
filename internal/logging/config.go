package logging

type Config struct {
	Level  string `yaml:"level"`  // "info", "debug", etc.
	Format string `yaml:"format"` // "json", "text"
}
