package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()

	if cfg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "claude-sonnet-4-20250514")
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens: got %d, want 4096", cfg.MaxTokens)
	}
	if cfg.DefaultFormat != "json" {
		t.Errorf("DefaultFormat: got %q, want %q", cfg.DefaultFormat, "json")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port: got %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host: got %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("RECAP_ANTHROPIC_KEY", "sk-ant-test-key-123")
	t.Setenv("RECAP_PORT", "9090")
	t.Setenv("RECAP_MODEL", "claude-opus-4-20250514")

	cfg := defaults()
	applyEnv(cfg)

	if cfg.AnthropicAPIKey != "sk-ant-test-key-123" {
		t.Errorf("AnthropicAPIKey: got %q, want %q", cfg.AnthropicAPIKey, "sk-ant-test-key-123")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port: got %d, want 9090", cfg.Server.Port)
	}
	if cfg.Model != "claude-opus-4-20250514" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "claude-opus-4-20250514")
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "empty key", key: "", want: "not set"},
		{name: "long key", key: "sk-ant-1234567890", want: "sk-ant-1..."},
		{name: "short key", key: "abc", want: "abc..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{AnthropicAPIKey: tt.key}
			redacted := Redact(cfg)
			if redacted.AnthropicAPIKey != tt.want {
				t.Errorf("Redact(%q): got %q, want %q", tt.key, redacted.AnthropicAPIKey, tt.want)
			}
		})
	}
}

func TestYAMLConfig(t *testing.T) {
	// Save and restore working directory.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	// Create a temp directory with .recap/config.yaml.
	tmpDir := t.TempDir()
	recapDir := filepath.Join(tmpDir, ".recap")
	if err := os.MkdirAll(recapDir, 0o755); err != nil {
		t.Fatalf("failed to create .recap dir: %v", err)
	}

	yamlContent := []byte("model: claude-haiku-4-20250514\nmax_tokens: 2048\n")
	if err := os.WriteFile(filepath.Join(recapDir, "config.yaml"), yamlContent, 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Change to the temp dir so CWD config is found.
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Model != "claude-haiku-4-20250514" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "claude-haiku-4-20250514")
	}
	if cfg.MaxTokens != 2048 {
		t.Errorf("MaxTokens: got %d, want 2048", cfg.MaxTokens)
	}

	// Defaults should still be intact for fields not in YAML.
	if cfg.DefaultFormat != "json" {
		t.Errorf("DefaultFormat: got %q, want %q", cfg.DefaultFormat, "json")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port: got %d, want 8080", cfg.Server.Port)
	}
}
