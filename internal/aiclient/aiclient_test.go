package aiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Provider inference tests ---

func TestInferProvider(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-sonnet-4-20250514", ProviderAnthropic},
		{"claude-opus-4-20250514", ProviderAnthropic},
		{"Claude-Haiku-4", ProviderAnthropic},
		{"gpt-4o", ProviderOpenAI},
		{"gpt-4o-mini", ProviderOpenAI},
		{"o3", ProviderOpenAI},
		{"o3-mini", ProviderOpenAI},
		{"o4-mini", ProviderOpenAI},
		{"gemini-2.5-pro", ProviderGoogle},
		{"gemini-2.0-flash", ProviderGoogle},
		{"unknown-model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := InferProvider(tt.model)
			if got != tt.want {
				t.Errorf("InferProvider(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestLookupModel(t *testing.T) {
	// Known model.
	info, ok := LookupModel("gpt-4o")
	if !ok {
		t.Fatal("expected to find gpt-4o in registry")
	}
	if info.Provider != ProviderOpenAI {
		t.Errorf("gpt-4o provider: got %q, want %q", info.Provider, ProviderOpenAI)
	}
	if info.Name != "GPT-4o" {
		t.Errorf("gpt-4o name: got %q, want %q", info.Name, "GPT-4o")
	}

	// Unknown model.
	_, ok = LookupModel("nonexistent-model")
	if ok {
		t.Error("expected nonexistent-model to not be found")
	}
}

func TestModelsByProvider(t *testing.T) {
	grouped := ModelsByProvider()

	for _, provider := range []string{ProviderAnthropic, ProviderOpenAI, ProviderGoogle} {
		models, ok := grouped[provider]
		if !ok || len(models) == 0 {
			t.Errorf("expected at least one model for provider %q", provider)
		}
	}
}

// --- Factory tests ---

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(ProviderConfig{
		Provider: ProviderAnthropic,
		APIKey:   "",
		Model:    "claude-sonnet-4-20250514",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNew_UnsupportedProvider(t *testing.T) {
	_, err := New(ProviderConfig{
		Provider: "unsupported",
		APIKey:   "some-key",
		Model:    "some-model",
	})
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestNew_AnthropicProvider(t *testing.T) {
	client, err := New(ProviderConfig{
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*AnthropicClient); !ok {
		t.Errorf("expected *AnthropicClient, got %T", client)
	}
}

func TestNew_OpenAIProvider(t *testing.T) {
	client, err := New(ProviderConfig{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*OpenAIClient); !ok {
		t.Errorf("expected *OpenAIClient, got %T", client)
	}
}

func TestNew_GoogleProvider(t *testing.T) {
	client, err := New(ProviderConfig{
		Provider: ProviderGoogle,
		APIKey:   "test-key",
		Model:    "gemini-2.5-pro",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*GoogleClient); !ok {
		t.Errorf("expected *GoogleClient, got %T", client)
	}
}

func TestNew_InfersProvider(t *testing.T) {
	client, err := New(ProviderConfig{
		APIKey: "test-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*OpenAIClient); !ok {
		t.Errorf("expected *OpenAIClient from inferred provider, got %T", client)
	}
}

// --- OpenAI mock server tests ---

func TestOpenAIClient_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected auth header 'Bearer test-key', got %q", auth)
		}

		resp := openaiResponse{
			Choices: []openaiChoice{
				{Message: openaiMessage{Role: "assistant", Content: `{"title": "Test"}`}},
			},
			Usage: openaiUsage{TotalTokens: 150},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OpenAIClient{
		apiKey:  "test-key",
		model:   "gpt-4o",
		baseURL: server.URL,
	}

	resp, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "You are a test assistant.",
		UserPrompt:   "Extract data.",
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Text != `{"title": "Test"}` {
		t.Errorf("text: got %q, want %q", resp.Text, `{"title": "Test"}`)
	}
	if resp.TokensUsed != 150 {
		t.Errorf("tokens: got %d, want 150", resp.TokensUsed)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("model: got %q, want %q", resp.Model, "gpt-4o")
	}
}

func TestOpenAIClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{
			Error: &openaiError{Message: "invalid api key"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OpenAIClient{
		apiKey:  "bad-key",
		model:   "gpt-4o",
		baseURL: server.URL,
	}

	_, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		UserPrompt:   "test",
		MaxTokens:    100,
	})
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestOpenAIClient_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{Choices: []openaiChoice{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &OpenAIClient{
		apiKey:  "test-key",
		model:   "gpt-4o",
		baseURL: server.URL,
	}

	_, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		UserPrompt:   "test",
		MaxTokens:    100,
	})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

// --- Google Gemini mock server tests ---

func TestGoogleClient_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{Content: geminiContent{Parts: []geminiPart{{Text: `{"title": "Gemini Test"}`}}}},
			},
			UsageMetadata: &geminiUsage{TotalTokenCount: 200},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &GoogleClient{
		apiKey:  "test-key",
		model:   "gemini-2.5-pro",
		baseURL: server.URL,
	}

	resp, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "You are a test assistant.",
		UserPrompt:   "Extract data.",
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Text != `{"title": "Gemini Test"}` {
		t.Errorf("text: got %q, want %q", resp.Text, `{"title": "Gemini Test"}`)
	}
	if resp.TokensUsed != 200 {
		t.Errorf("tokens: got %d, want 200", resp.TokensUsed)
	}
	if resp.Model != "gemini-2.5-pro" {
		t.Errorf("model: got %q, want %q", resp.Model, "gemini-2.5-pro")
	}
}

func TestGoogleClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Error: &geminiError{Message: "invalid key", Code: 401},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &GoogleClient{
		apiKey:  "bad-key",
		model:   "gemini-2.5-pro",
		baseURL: server.URL,
	}

	_, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		UserPrompt:   "test",
		MaxTokens:    100,
	})
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestGoogleClient_NoCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{Candidates: []geminiCandidate{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &GoogleClient{
		apiKey:  "test-key",
		model:   "gemini-2.5-pro",
		baseURL: server.URL,
	}

	_, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "test",
		UserPrompt:   "test",
		MaxTokens:    100,
	})
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
}
