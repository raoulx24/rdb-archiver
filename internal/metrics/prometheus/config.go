package prometheus

import (
	"log"
	"slices"
)

type Config struct {
	Enabled          bool      `yaml:"enabled"`
	Port             uint16    `yaml:"port"`
	HistogramBuckets []float64 `yaml:"histogramBuckets"`
}

func (c *Config) ApplyDefaults(logger *log.Logger) {
	if c.Port == 0 {
		c.Port = 9090
	}

	b := validateBuckets(c.HistogramBuckets)

	if !slices.Equal(b, c.HistogramBuckets) {
		if logger != nil {
			logger.Printf("WARN: invalid histogram buckets; using defaults (provided=%v fallback=%v)", c.HistogramBuckets, b)
		}
	}

	c.HistogramBuckets = b
}

func validateBuckets(b []float64) []float64 {
	defaultBuckets := []float64{0.1, 0.5, 1, 5}

	if len(b) == 0 {
		return defaultBuckets
	}
	if b[0] <= 0 {
		return defaultBuckets
	}
	for i := 1; i < len(b); i++ {
		if b[i] <= b[i-1] {
			return defaultBuckets
		}
	}
	return b
}
