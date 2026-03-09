package aiclient

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClient implements AIClient using the Anthropic Messages API.
type AnthropicClient struct {
	client anthropic.Client
	model  string
}

// NewAnthropicClient creates a new AnthropicClient.
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Complete sends a completion request to the Anthropic API.
func (c *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: int64(req.MaxTokens),
		System: []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserPrompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API call failed: %w", err)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("anthropic API returned empty response")
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	if text == "" {
		return nil, fmt.Errorf("anthropic API returned no text block")
	}

	return &CompletionResponse{
		Text:       text,
		TokensUsed: int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		Model:      c.model,
	}, nil
}
