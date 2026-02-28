package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// matches $(VAR_NAME)
var envPattern = regexp.MustCompile(`\$\(([A-Za-z0-9_]+)\)`)

// replaces $(VAR) with os.Getenv(VAR)
func expandEnvVars(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(m string) string {
		key := mapEnvKey(envPattern.FindStringSubmatch(m)[1])
		return os.Getenv(key)
	})
}

func Load(path string) (*Config, error) {
	// read raw YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// expand $(ENV_VAR) placeholders
	expanded := expandEnvVars(string(data))

	// unmarshal into struct
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling yaml: %w", err)
	}

	return &cfg, nil
}
