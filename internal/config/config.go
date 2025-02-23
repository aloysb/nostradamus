package config

import (
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	Debug bool
}

// New creates a new Config instance
func New() *Config {
	return &Config{
		Debug: os.Getenv("DEBUG") == "1",
	}
}

// RetryDelay defines the waiting period between API call attempts.
var RetryDelay = 1 * time.Second
