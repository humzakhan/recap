package extractor

import (
	"strings"
	"testing"
	"time"
)

func TestBuildPrompt(t *testing.T) {
	content := FetchedContent{
		URL:         "https://example.com/article",
		ContentType: "article",
		Title:       "Test Article",
		RawText:     "This is the article body text for testing purposes.",
		FetchedAt:   time.Now(),
	}

	fields := []Field{
		{Label: "Author Name", Type: FieldTypeText},
		{Label: "Key Takeaways", Type: FieldTypeList},
		{Label: "sentiment", Type: FieldTypeEnum, Options: []string{"positive", "negative", "neutral"}},
	}

	systemPrompt, userPrompt := BuildPrompt(content, fields)

	t.Run("system prompt matches constant", func(t *testing.T) {
		if systemPrompt != SystemPrompt {
			t.Errorf("system prompt does not match SystemPrompt constant")
		}
	})

	t.Run("user prompt starts with Schema", func(t *testing.T) {
		if !strings.HasPrefix(userPrompt, "Schema:\n") {
			t.Errorf("user prompt does not start with \"Schema:\\n\", got prefix: %q", userPrompt[:min(len(userPrompt), 30)])
		}
	})

	t.Run("user prompt contains Content section", func(t *testing.T) {
		if !strings.Contains(userPrompt, "Content:\n") {
			t.Error("user prompt does not contain \"Content:\\n\"")
		}
	})

	t.Run("fields have snake_case labels", func(t *testing.T) {
		if !strings.Contains(userPrompt, "author_name") {
			t.Error("user prompt does not contain snake_case label \"author_name\"")
		}
		if !strings.Contains(userPrompt, "key_takeaways") {
			t.Error("user prompt does not contain snake_case label \"key_takeaways\"")
		}
		if !strings.Contains(userPrompt, "sentiment") {
			t.Error("user prompt does not contain label \"sentiment\"")
		}
	})

	t.Run("enum options appear in JSON", func(t *testing.T) {
		if !strings.Contains(userPrompt, "positive") {
			t.Error("user prompt does not contain enum option \"positive\"")
		}
		if !strings.Contains(userPrompt, "negative") {
			t.Error("user prompt does not contain enum option \"negative\"")
		}
		if !strings.Contains(userPrompt, "neutral") {
			t.Error("user prompt does not contain enum option \"neutral\"")
		}
	})
}

func TestTruncateContent(t *testing.T) {
	t.Run("short text unchanged", func(t *testing.T) {
		text := "This is short content."
		got := TruncateContent(text, DefaultMaxContentChars)
		if got != text {
			t.Errorf("short text was modified: got length %d, want %d", len(got), len(text))
		}
	})

	t.Run("long text is truncated to approximately maxChars", func(t *testing.T) {
		text := strings.Repeat("x", 100000)
		got := TruncateContent(text, DefaultMaxContentChars)
		if len(got) > DefaultMaxContentChars {
			t.Errorf("truncated length %d exceeds max %d", len(got), DefaultMaxContentChars)
		}
	})

	t.Run("truncated text contains marker", func(t *testing.T) {
		text := strings.Repeat("a", 100000)
		got := TruncateContent(text, DefaultMaxContentChars)
		if !strings.Contains(got, "[...content truncated...]") {
			t.Error("truncated text does not contain truncation marker")
		}
	})

	t.Run("beginning of content is preserved", func(t *testing.T) {
		prefix := "START_OF_CONTENT_"
		text := prefix + strings.Repeat("m", 100000)
		got := TruncateContent(text, DefaultMaxContentChars)
		if !strings.HasPrefix(got, prefix) {
			t.Error("truncated text does not preserve the beginning of the content")
		}
	})

	t.Run("end of content is preserved", func(t *testing.T) {
		suffix := "_END_OF_CONTENT"
		text := strings.Repeat("m", 100000) + suffix
		got := TruncateContent(text, DefaultMaxContentChars)
		if !strings.HasSuffix(got, suffix) {
			t.Error("truncated text does not preserve the end of the content")
		}
	})
}
