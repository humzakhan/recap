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
	if cfg.APIKeys == nil {
		t.Error("APIKeys: expected non-nil map")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("RECAP_ANTHROPIC_KEY", "sk-ant-test-key-123")
	t.Setenv("RECAP_OPENAI_KEY", "sk-openai-test-key")
	t.Setenv("RECAP_GOOGLE_KEY", "AIza-google-test-key")
	t.Setenv("RECAP_PORT", "9090")
	t.Setenv("RECAP_MODEL", "claude-opus-4-20250514")

	cfg := defaults()
	applyEnv(cfg)

	if cfg.AnthropicAPIKey != "sk-ant-test-key-123" {
		t.Errorf("AnthropicAPIKey: got %q, want %q", cfg.AnthropicAPIKey, "sk-ant-test-key-123")
	}
	if cfg.APIKeys["anthropic"] != "sk-ant-test-key-123" {
		t.Errorf("APIKeys[anthropic]: got %q, want %q", cfg.APIKeys["anthropic"], "sk-ant-test-key-123")
	}
	if cfg.APIKeys["openai"] != "sk-openai-test-key" {
		t.Errorf("APIKeys[openai]: got %q, want %q", cfg.APIKeys["openai"], "sk-openai-test-key")
	}
	if cfg.APIKeys["google"] != "AIza-google-test-key" {
		t.Errorf("APIKeys[google]: got %q, want %q", cfg.APIKeys["google"], "AIza-google-test-key")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port: got %d, want 9090", cfg.Server.Port)
	}
	if cfg.Model != "claude-opus-4-20250514" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "claude-opus-4-20250514")
	}
}

func TestProviderAPIKey(t *testing.T) {
	// Test APIKeys map lookup.
	cfg := &Config{
		APIKeys: map[string]string{
			"anthropic": "key-from-map",
			"openai":    "openai-key",
		},
	}
	if got := cfg.ProviderAPIKey("anthropic"); got != "key-from-map" {
		t.Errorf("ProviderAPIKey(anthropic): got %q, want %q", got, "key-from-map")
	}
	if got := cfg.ProviderAPIKey("openai"); got != "openai-key" {
		t.Errorf("ProviderAPIKey(openai): got %q, want %q", got, "openai-key")
	}
	if got := cfg.ProviderAPIKey("google"); got != "" {
		t.Errorf("ProviderAPIKey(google): got %q, want empty", got)
	}

	// Test legacy fallback for anthropic.
	cfgLegacy := &Config{
		AnthropicAPIKey: "legacy-key",
	}
	if got := cfgLegacy.ProviderAPIKey("anthropic"); got != "legacy-key" {
		t.Errorf("ProviderAPIKey(anthropic) legacy: got %q, want %q", got, "legacy-key")
	}
	if got := cfgLegacy.ProviderAPIKey("openai"); got != "" {
		t.Errorf("ProviderAPIKey(openai) legacy: got %q, want empty", got)
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

func TestRedact_APIKeys(t *testing.T) {
	cfg := &Config{
		APIKeys: map[string]string{
			"anthropic": "sk-ant-1234567890",
			"openai":    "sk-openai-abcdefgh",
			"google":    "",
		},
	}
	redacted := Redact(cfg)

	if redacted.APIKeys["anthropic"] != "sk-ant-1..." {
		t.Errorf("APIKeys[anthropic]: got %q, want %q", redacted.APIKeys["anthropic"], "sk-ant-1...")
	}
	if redacted.APIKeys["openai"] != "sk-opena..." {
		t.Errorf("APIKeys[openai]: got %q, want %q", redacted.APIKeys["openai"], "sk-opena...")
	}
	if redacted.APIKeys["google"] != "not set" {
		t.Errorf("APIKeys[google]: got %q, want %q", redacted.APIKeys["google"], "not set")
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

func TestYAMLConfig_LegacyMigration(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	tmpDir := t.TempDir()
	recapDir := filepath.Join(tmpDir, ".recap")
	if err := os.MkdirAll(recapDir, 0o755); err != nil {
		t.Fatalf("failed to create .recap dir: %v", err)
	}

	// Use legacy anthropic_api_key field.
	yamlContent := []byte("anthropic_api_key: sk-ant-legacy-key\n")
	if err := os.WriteFile(filepath.Join(recapDir, "config.yaml"), yamlContent, 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Legacy key should be migrated to APIKeys map.
	if cfg.ProviderAPIKey("anthropic") != "sk-ant-legacy-key" {
		t.Errorf("ProviderAPIKey(anthropic): got %q, want %q", cfg.ProviderAPIKey("anthropic"), "sk-ant-legacy-key")
	}
	if cfg.APIKeys["anthropic"] != "sk-ant-legacy-key" {
		t.Errorf("APIKeys[anthropic]: got %q, want %q", cfg.APIKeys["anthropic"], "sk-ant-legacy-key")
	}
}

func TestYAMLConfig_APIKeysMap(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	tmpDir := t.TempDir()
	recapDir := filepath.Join(tmpDir, ".recap")
	if err := os.MkdirAll(recapDir, 0o755); err != nil {
		t.Fatalf("failed to create .recap dir: %v", err)
	}

	yamlContent := []byte(`model: gpt-4o
api_keys:
  openai: sk-openai-from-yaml
  google: AIza-google-from-yaml
`)
	if err := os.WriteFile(filepath.Join(recapDir, "config.yaml"), yamlContent, 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Model != "gpt-4o" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "gpt-4o")
	}
	if cfg.ProviderAPIKey("openai") != "sk-openai-from-yaml" {
		t.Errorf("ProviderAPIKey(openai): got %q, want %q", cfg.ProviderAPIKey("openai"), "sk-openai-from-yaml")
	}
	if cfg.ProviderAPIKey("google") != "AIza-google-from-yaml" {
		t.Errorf("ProviderAPIKey(google): got %q, want %q", cfg.ProviderAPIKey("google"), "AIza-google-from-yaml")
	}
}
