package config

import (
	"crypto/sha256"
	"encoding/hex"
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

func Load(path string) (*Config, string, error) {
	// read raw YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("reading config file: %w", err)
	}

	hash := SHA256Bytes(data)

	// expand $(ENV_VAR) placeholders
	expanded := expandEnvVars(string(data))

	// unmarshal into struct
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, "", fmt.Errorf("unmarshalling yaml: %w", err)
	}

	return &cfg, hash, nil
}

func (c *Config) ApplyDefaults() {
	c.Source.ApplyDefaults()
	c.Destination.ApplyDefaults()
	c.WatchFS.ApplyDefaults()
	c.FS.ApplyDefaults()
	c.Logging.ApplyDefaults()
	c.Health.ApplyDefaults()
	c.ConfigReload.ApplyDefaults()
}

func (rc *ReloadConfig) ApplyDefaults() {
	if rc.Method == "" {
		rc.Method = "fsnotify"
	}
}

func SHA256Bytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
