package config

import "os"

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
