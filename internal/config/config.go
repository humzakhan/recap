package config

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	AnthropicAPIKey string       `yaml:"anthropic_api_key"`
	Model           string       `yaml:"model"`
	MaxTokens       int          `yaml:"max_tokens"`
	DefaultFormat   string       `yaml:"default_format"`
	Server          ServerConfig `yaml:"server"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// defaults returns a Config populated with default values.
func defaults() *Config {
	return &Config{
		Model:         "claude-sonnet-4-20250514",
		MaxTokens:     4096,
		DefaultFormat: "json",
		Server: ServerConfig{
			Port: 8080,
			Host: "127.0.0.1",
		},
	}
}

// Load loads configuration with layered priority:
// 1. Defaults
// 2. ~/.recap/config.yaml (if exists)
// 3. .recap/config.yaml in CWD (if exists)
// 4. Environment variables (highest priority)
func Load() (*Config, error) {
	cfg := defaults()

	// Load from ~/.recap/config.yaml
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".recap", "config.yaml")
		if err := loadYAML(homeConfig, cfg); err != nil {
			return nil, err
		}
	}

	// Load from .recap/config.yaml in CWD
	cwd, err := os.Getwd()
	if err == nil {
		cwdConfig := filepath.Join(cwd, ".recap", "config.yaml")
		if err := loadYAML(cwdConfig, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	applyEnv(cfg)

	return cfg, nil
}

// loadYAML reads a YAML file and merges non-zero values into cfg.
func loadYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read — skip silently.
		return nil
	}

	var tmp Config
	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return err
	}

	merge(cfg, &tmp)
	return nil
}

// merge copies non-zero values from src into dst.
func merge(dst, src *Config) {
	if src.AnthropicAPIKey != "" {
		dst.AnthropicAPIKey = src.AnthropicAPIKey
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.MaxTokens != 0 {
		dst.MaxTokens = src.MaxTokens
	}
	if src.DefaultFormat != "" {
		dst.DefaultFormat = src.DefaultFormat
	}
	if src.Server.Port != 0 {
		dst.Server.Port = src.Server.Port
	}
	if src.Server.Host != "" {
		dst.Server.Host = src.Server.Host
	}
}

// applyEnv overrides config values from environment variables.
func applyEnv(cfg *Config) {
	if v := os.Getenv("RECAP_ANTHROPIC_KEY"); v != "" {
		cfg.AnthropicAPIKey = v
	}
	if v := os.Getenv("RECAP_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("RECAP_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxTokens = n
		}
	}
	if v := os.Getenv("RECAP_DEFAULT_FORMAT"); v != "" {
		cfg.DefaultFormat = v
	}
	if v := os.Getenv("RECAP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Server.Port = n
		}
	}
	if v := os.Getenv("RECAP_HOST"); v != "" {
		cfg.Server.Host = v
	}
}

// Redact returns a copy of the config with the API key masked for safe display.
// Shows the first 8 characters followed by "..." or "not set" if the key is empty.
func Redact(cfg *Config) Config {
	redacted := *cfg
	if redacted.AnthropicAPIKey == "" {
		redacted.AnthropicAPIKey = "not set"
	} else if len(redacted.AnthropicAPIKey) > 8 {
		redacted.AnthropicAPIKey = redacted.AnthropicAPIKey[:8] + "..."
	} else {
		redacted.AnthropicAPIKey = redacted.AnthropicAPIKey + "..."
	}
	return redacted
}
