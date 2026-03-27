package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the css-trimmer configuration file structure.
type Config struct {
	Whitelist            []string `yaml:"whitelist"`
	Blacklist            []string `yaml:"blacklist"`
	Extensions           []string `yaml:"extensions"`
	DynamicClassPatterns []string `yaml:"dynamic_class_patterns"`
	FailOnRemoval        bool     `yaml:"fail_on_removal"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Whitelist:            []string{},
		Blacklist:            []string{},
		Extensions:           []string{".html", ".templ", ".jsx", ".tsx"},
		DynamicClassPatterns: []string{},
		FailOnRemoval:        false,
	}
}

// Load reads and parses a YAML config file, merging with defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// If file doesn't exist, return defaults
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure extensions start with a dot
	for i, ext := range cfg.Extensions {
		if len(ext) > 0 && ext[0] != '.' {
			cfg.Extensions[i] = "." + ext
		}
	}

	return cfg, nil
}

// IsExtensionIncluded checks if a file extension should be scanned.
func (c *Config) IsExtensionIncluded(filePath string) bool {
	ext := filepath.Ext(filePath)
	for _, e := range c.Extensions {
		if e == ext {
			return true
		}
	}
	return false
}
