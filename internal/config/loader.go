package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads an lzctl.yaml file, parses it into an LZConfig struct,
// and applies default values for optional fields.
func Load(path string) (*LZConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse parses raw YAML bytes into an LZConfig struct and applies defaults.
func Parse(data []byte) (*LZConfig, error) {
	var cfg LZConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config YAML: %w", err)
	}
	ApplyDefaults(&cfg)
	return &cfg, nil
}
