package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Fetcher is the interface for fetching URL content, allowing test mocks.
type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (*FetchedContent, error)
}

// Extractor holds the configuration needed to perform an extraction.
type Extractor struct {
	Fetcher   Fetcher
	APIKey    string
	Model     string
	MaxTokens int
}

// NewExtractor creates a new Extractor with the given dependencies and config.
func NewExtractor(fetcher Fetcher, apiKey, model string, maxTokens int) *Extractor {
	return &Extractor{
		Fetcher:   fetcher,
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: maxTokens,
	}
}

// Extract fetches content from the given URL, sends it to the LLM with the
// requested fields, and returns the structured extraction result.
func (e *Extractor) Extract(ctx context.Context, req ExtractionRequest) (*ExtractionResult, error) {
	// 1. Fetch content from the URL.
	content, err := e.Fetcher.Fetch(ctx, req.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// 2. Build the LLM prompts.
	systemPrompt, userPrompt := BuildPrompt(*content, req.Fields)

	// 3. Call the Anthropic API.
	client := anthropic.NewClient(option.WithAPIKey(e.APIKey))
	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(e.Model),
		MaxTokens: int64(e.MaxTokens),
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// 4. Extract text from the response.
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("extraction failed: empty response from model")
	}

	var responseText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}
	if responseText == "" {
		return nil, fmt.Errorf("extraction failed: no text block in model response")
	}

	// 5. Parse and validate the response.
	data, err := parseResponse(responseText, req.Fields)
	if err != nil {
		return nil, err
	}

	// 6. Build the result.
	return &ExtractionResult{
		URL:         req.URL,
		ContentType: content.ContentType,
		ExtractedAt: time.Now(),
		Data:        data,
		RawContent:  content.RawText,
		Model:       e.Model,
		TokensUsed:  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
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
		return nil, fmt.Errorf("failed to parse model response: %w", err)
	}

	result := make(map[string]any, len(fields))

	for _, f := range fields {
		key := ToSnakeCase(f.Label)
		val, exists := raw[key]
		if !exists {
			result[key] = nil
			continue
		}

		coerced, err := CoerceValue(val, f.Type, f.Options)
		if err != nil {
			log.Printf("warning: coercion failed for field %q (type %s): %v; setting to null", key, f.Type, err)
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
