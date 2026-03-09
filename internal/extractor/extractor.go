package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/humzakhan/recap/internal/aiclient"
	"github.com/humzakhan/recap/internal/log"
)

// Fetcher is the interface for fetching URL content, allowing test mocks.
type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (*FetchedContent, error)
}

// Extractor holds the configuration needed to perform an extraction.
type Extractor struct {
	Fetcher   Fetcher
	Client    aiclient.AIClient
	MaxTokens int
}

// NewExtractor creates a new Extractor with the given dependencies.
func NewExtractor(fetcher Fetcher, client aiclient.AIClient, maxTokens int) *Extractor {
	return &Extractor{
		Fetcher:   fetcher,
		Client:    client,
		MaxTokens: maxTokens,
	}
}

// Extract fetches content from the given URL, sends it to the LLM with the
// requested fields, and returns the structured extraction result.
func (e *Extractor) Extract(ctx context.Context, req ExtractionRequest) (*ExtractionResult, error) {
	// 1. Fetch content from the URL.
	log.Debug("extractor: fetching content from %q", req.URL)
	content, err := e.Fetcher.Fetch(ctx, req.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed for %q: %w", req.URL, err)
	}
	log.Debug("extractor: fetched %d chars of %s content (title: %q)", len(content.RawText), content.ContentType, content.Title)

	// 2. Build the LLM prompts.
	log.Debug("extractor: building prompt for %d fields", len(req.Fields))
	systemPrompt, userPrompt := BuildPrompt(*content, req.Fields)
	log.Debug("extractor: system prompt %d chars, user prompt %d chars", len(systemPrompt), len(userPrompt))

	// 3. Call the AI provider.
	log.Debug("extractor: calling AI provider (max_tokens=%d)", e.MaxTokens)
	startTime := time.Now()
	resp, err := e.Client.Complete(ctx, aiclient.CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    e.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}
	log.Debug("extractor: AI response received in %s (model=%s, tokens=%d)", time.Since(startTime).Round(time.Millisecond), resp.Model, resp.TokensUsed)
	log.Debug("extractor: raw response (%d chars): %.200s", len(resp.Text), resp.Text)

	// 4. Parse and validate the response.
	log.Debug("extractor: parsing and validating response")
	data, err := parseResponse(resp.Text, req.Fields)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model output: %w\nRaw response: %.500s", err, resp.Text)
	}
	log.Debug("extractor: successfully extracted %d fields", len(data))

	// 5. Build the result.
	return &ExtractionResult{
		URL:         req.URL,
		ContentType: content.ContentType,
		ExtractedAt: time.Now(),
		Data:        data,
		RawContent:  content.RawText,
		Model:       resp.Model,
		TokensUsed:  resp.TokensUsed,
	}, nil
}

// parseResponse parses the model's text output as JSON and validates/coerces
// each field value against its declared type. Missing keys are set to nil.
// Coercion warnings are logged but do not cause an error.
func parseResponse(text string, fields []Field) (map[string]any, error) {
	// Strip markdown code fences if present.
	cleaned := stripCodeFences(text)

	var raw map[string]any
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result := make(map[string]any, len(fields))

	for _, f := range fields {
		key := ToSnakeCase(f.Label)
		val, exists := raw[key]
		if !exists {
			log.Debug("extractor: field %q not found in response, setting to null", key)
			result[key] = nil
			continue
		}

		coerced, err := CoerceValue(val, f.Type, f.Options)
		if err != nil {
			log.Warn("coercion failed for field %q (type %s): %v; setting to null", key, f.Type, err)
			result[key] = nil
			continue
		}
		result[key] = coerced
	}

	return result, nil
}

// stripCodeFences removes markdown code fences (```json ... ``` or ``` ... ```)
// from the text if present, returning only the inner content.
func stripCodeFences(text string) string {
	trimmed := strings.TrimSpace(text)

	// Check for ```json or ``` prefix.
	if strings.HasPrefix(trimmed, "```") {
		// Find the end of the opening fence line.
		firstNewline := strings.Index(trimmed, "\n")
		if firstNewline == -1 {
			return trimmed
		}

		// Find the closing fence.
		lastFence := strings.LastIndex(trimmed, "```")
		if lastFence > firstNewline {
			return strings.TrimSpace(trimmed[firstNewline+1 : lastFence])
		}
	}

	return trimmed
}
