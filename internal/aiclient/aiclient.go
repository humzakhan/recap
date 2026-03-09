// Package aiclient provides a provider-agnostic interface for LLM completions.
package aiclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/humzakhan/recap/internal/log"
)

// AIClient is the interface that all LLM providers must implement.
type AIClient interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// CompletionRequest contains the inputs for an LLM completion call.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
}

// CompletionResponse contains the outputs from an LLM completion call.
type CompletionResponse struct {
	Text       string
	TokensUsed int
	Model      string
}

// ProviderConfig holds the configuration needed to create an AIClient.
type ProviderConfig struct {
	Provider string
	APIKey   string
	Model    string
}

// New creates an AIClient for the given configuration. If Provider is empty,
// it is inferred from the Model name using the model registry.
func New(cfg ProviderConfig) (AIClient, error) {
	provider := cfg.Provider
	if provider == "" {
		info, ok := LookupModel(cfg.Model)
		if ok {
			provider = info.Provider
		} else {
			provider = InferProvider(cfg.Model)
		}
	}

	if cfg.APIKey == "" {
		hint := ""
		switch provider {
		case ProviderAnthropic:
			hint = "\n  Set RECAP_ANTHROPIC_KEY or add api_keys.anthropic to ~/.recap/config.yaml"
		case ProviderOpenAI:
			hint = "\n  Set RECAP_OPENAI_KEY or add api_keys.openai to ~/.recap/config.yaml"
		case ProviderGoogle:
			hint = "\n  Set RECAP_GOOGLE_KEY or add api_keys.google to ~/.recap/config.yaml"
		}
		return nil, fmt.Errorf("no API key configured for provider %q%s", provider, hint)
	}

	log.Debug("aiclient: creating %s client for model %q", provider, cfg.Model)

	switch strings.ToLower(provider) {
	case ProviderAnthropic:
		return NewAnthropicClient(cfg.APIKey, cfg.Model), nil
	case ProviderOpenAI:
		return NewOpenAIClient(cfg.APIKey, cfg.Model), nil
	case ProviderGoogle:
		return NewGoogleClient(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %q\n  Supported providers: anthropic, openai, google\n  Run 'recap config models' to see available models", provider)
	}
}

// InferProvider guesses the provider from a model ID string when the model
// is not found in the registry. This allows users to use new models that
// haven't been added to the registry yet.
func InferProvider(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "claude"):
		return ProviderAnthropic
	case strings.HasPrefix(m, "gpt"), strings.HasPrefix(m, "o1"), strings.HasPrefix(m, "o3"), strings.HasPrefix(m, "o4"):
		return ProviderOpenAI
	case strings.HasPrefix(m, "gemini"):
		return ProviderGoogle
	default:
		return ""
	}
}
