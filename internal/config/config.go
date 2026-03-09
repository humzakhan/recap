package config

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	// Deprecated: use APIKeys["anthropic"] instead. Kept for backward compat
	// when reading existing config files.
	AnthropicAPIKey string `yaml:"anthropic_api_key,omitempty"`

	Model         string            `yaml:"model"`
	MaxTokens     int               `yaml:"max_tokens"`
	DefaultFormat string            `yaml:"default_format"`
	Server        ServerConfig      `yaml:"server"`
	APIKeys       map[string]string `yaml:"api_keys,omitempty"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// ProviderAPIKey returns the API key for the given provider, checking the
// APIKeys map first, then falling back to the legacy AnthropicAPIKey field.
func (c *Config) ProviderAPIKey(provider string) string {
	if c.APIKeys != nil {
		if key, ok := c.APIKeys[provider]; ok && key != "" {
			return key
		}
	}
	// Legacy fallback: if looking up anthropic and only the old field is set.
	if provider == "anthropic" && c.AnthropicAPIKey != "" {
		return c.AnthropicAPIKey
	}
	return ""
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
		APIKeys: make(map[string]string),
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

	// Migrate legacy anthropic_api_key into APIKeys map.
	if cfg.AnthropicAPIKey != "" && cfg.ProviderAPIKey("anthropic") == cfg.AnthropicAPIKey {
		if cfg.APIKeys == nil {
			cfg.APIKeys = make(map[string]string)
		}
		if cfg.APIKeys["anthropic"] == "" {
			cfg.APIKeys["anthropic"] = cfg.AnthropicAPIKey
		}
	}

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
	for k, v := range src.APIKeys {
		if v != "" {
			if dst.APIKeys == nil {
				dst.APIKeys = make(map[string]string)
			}
			dst.APIKeys[k] = v
		}
	}
}

// applyEnv overrides config values from environment variables.
func applyEnv(cfg *Config) {
	if v := os.Getenv("RECAP_ANTHROPIC_KEY"); v != "" {
		cfg.AnthropicAPIKey = v
		if cfg.APIKeys == nil {
			cfg.APIKeys = make(map[string]string)
		}
		cfg.APIKeys["anthropic"] = v
	}
	if v := os.Getenv("RECAP_OPENAI_KEY"); v != "" {
		if cfg.APIKeys == nil {
			cfg.APIKeys = make(map[string]string)
		}
		cfg.APIKeys["openai"] = v
	}
	if v := os.Getenv("RECAP_GOOGLE_KEY"); v != "" {
		if cfg.APIKeys == nil {
			cfg.APIKeys = make(map[string]string)
		}
		cfg.APIKeys["google"] = v
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

// Redact returns a copy of the config with API keys masked for safe display.
func Redact(cfg *Config) Config {
	redacted := *cfg
	redacted.AnthropicAPIKey = redactKey(cfg.AnthropicAPIKey)

	if len(cfg.APIKeys) > 0 {
		redacted.APIKeys = make(map[string]string, len(cfg.APIKeys))
		for k, v := range cfg.APIKeys {
			redacted.APIKeys[k] = redactKey(v)
		}
	}

	return redacted
}

// redactKey masks an API key for safe display.
func redactKey(key string) string {
	if key == "" {
		return "not set"
	}
	if len(key) > 8 {
		return key[:8] + "..."
	}
	return key + "..."
}
