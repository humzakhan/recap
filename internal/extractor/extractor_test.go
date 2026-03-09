package extractor

import (
	"strings"
	"testing"
)

// allFields is the standard set of fields used across parseResponse tests.
var allFields = []Field{
	{Label: "title", Type: FieldTypeText},
	{Label: "tags", Type: FieldTypeList},
	{Label: "sentiment", Type: FieldTypeEnum, Options: []string{"positive", "negative", "neutral"}},
	{Label: "word_count", Type: FieldTypeNumber},
	{Label: "is_draft", Type: FieldTypeBoolean},
	{Label: "published_at", Type: FieldTypeDate},
}

func TestParseResponse_ValidJSON(t *testing.T) {
	input := `{"title": "Test Article", "tags": ["go", "cli"], "sentiment": "positive", "word_count": 1500, "is_draft": false, "published_at": "2024-01-15"}`

	data, err := parseResponse(input, allFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check title (text).
	if v, ok := data["title"].(string); !ok || v != "Test Article" {
		t.Errorf("title: got %v, want %q", data["title"], "Test Article")
	}

	// Check tags (list).
	tags, ok := data["tags"].([]any)
	if !ok {
		t.Fatalf("tags: expected []any, got %T", data["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("tags: got %d items, want 2", len(tags))
	}

	// Check sentiment (enum).
	if v, ok := data["sentiment"].(string); !ok || v != "positive" {
		t.Errorf("sentiment: got %v, want %q", data["sentiment"], "positive")
	}

	// Check word_count (number).
	if v, ok := data["word_count"].(float64); !ok || v != 1500 {
		t.Errorf("word_count: got %v, want 1500", data["word_count"])
	}

	// Check is_draft (boolean).
	if v, ok := data["is_draft"].(bool); !ok || v != false {
		t.Errorf("is_draft: got %v, want false", data["is_draft"])
	}

	// Check published_at (date).
	if v, ok := data["published_at"].(string); !ok || v != "2024-01-15" {
		t.Errorf("published_at: got %v, want %q", data["published_at"], "2024-01-15")
	}
}

func TestParseResponse_WithCodeFences(t *testing.T) {
	input := "```json\n{\"title\": \"Test\"}\n```"

	fields := []Field{
		{Label: "title", Type: FieldTypeText},
	}

	data, err := parseResponse(input, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v, ok := data["title"].(string); !ok || v != "Test" {
		t.Errorf("title: got %v, want %q", data["title"], "Test")
	}
}

func TestParseResponse_MissingKeys(t *testing.T) {
	input := `{"title": "Test"}`

	fields := []Field{
		{Label: "title", Type: FieldTypeText},
		{Label: "author", Type: FieldTypeText},
	}

	data, err := parseResponse(input, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v, ok := data["title"].(string); !ok || v != "Test" {
		t.Errorf("title: got %v, want %q", data["title"], "Test")
	}

	if data["author"] != nil {
		t.Errorf("author: got %v, want nil", data["author"])
	}
}

func TestParseResponse_WrongTypeCoerced(t *testing.T) {
	// word_count is a number but we pass it where text is expected.
	input := `{"title": 42}`

	fields := []Field{
		{Label: "title", Type: FieldTypeText},
	}

	data, err := parseResponse(input, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Number should be coerced to string for a text field.
	v, ok := data["title"].(string)
	if !ok {
		t.Fatalf("title: expected string after coercion, got %T (%v)", data["title"], data["title"])
	}
	if v != "42" {
		t.Errorf("title: got %q, want %q", v, "42")
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	input := `{not valid json`

	_, err := parseResponse(input, allFields)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse model response") {
		t.Errorf("error should contain %q, got: %v", "failed to parse model response", err)
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON no fences",
			input: `  {"title": "Test"}  `,
			want:  `{"title": "Test"}`,
		},
		{
			name:  "json fence",
			input: "```json\n{\"title\": \"Test\"}\n```",
			want:  `{"title": "Test"}`,
		},
		{
			name:  "bare fence",
			input: "```\n{\"title\": \"Test\"}\n```",
			want:  `{"title": "Test"}`,
		},
		{
			name:  "no closing fence",
			input: "```json\n{\"title\": \"Test\"}",
			want:  "```json\n{\"title\": \"Test\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFences(tt.input)
			if got != tt.want {
				t.Errorf("stripCodeFences(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.want)
			}
		})
	}
}
