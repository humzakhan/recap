package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GoogleClient implements AIClient using the Google Gemini API.
type GoogleClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewGoogleClient creates a new GoogleClient.
func NewGoogleClient(apiKey, model string) *GoogleClient {
	return &GoogleClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}

// Gemini API request types.
type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// Gemini API response types.
type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage   `json:"usageMetadata,omitempty"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiUsage struct {
	TotalTokenCount int `json:"totalTokenCount"`
}

type geminiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Complete sends a completion request to the Gemini API.
func (c *GoogleClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	body := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		},
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: req.UserPrompt}}},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var resp geminiResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s", resp.Error.Message)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini API returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini API returned no content")
	}

	tokensUsed := 0
	if resp.UsageMetadata != nil {
		tokensUsed = resp.UsageMetadata.TotalTokenCount
	}

	return &CompletionResponse{
		Text:       resp.Candidates[0].Content.Parts[0].Text,
		TokensUsed: tokensUsed,
		Model:      c.model,
	}, nil
}
