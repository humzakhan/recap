package main

import (
	"strings"
	"testing"
	"time"

	"github.com/humzakhan/recap/internal/extractor"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   bool
		check     func(t *testing.T, fields []extractor.Field)
	}{
		{
			name:      "bare label defaults to text",
			input:     "author",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Label != "author" {
					t.Errorf("expected Label \"author\", got %q", fields[0].Label)
				}
				if fields[0].Type != extractor.FieldTypeText {
					t.Errorf("expected Type text, got %q", fields[0].Type)
				}
			},
		},
		{
			name:      "explicit text type",
			input:     "author:text",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Label != "author" {
					t.Errorf("expected Label \"author\", got %q", fields[0].Label)
				}
				if fields[0].Type != extractor.FieldTypeText {
					t.Errorf("expected Type text, got %q", fields[0].Type)
				}
			},
		},
		{
			name:      "list type",
			input:     "takeaways:list",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Type != extractor.FieldTypeList {
					t.Errorf("expected Type list, got %q", fields[0].Type)
				}
			},
		},
		{
			name:      "enum with single option (ParseFields splits on comma)",
			input:     "status:enum[draft]",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Type != extractor.FieldTypeEnum {
					t.Errorf("expected Type enum, got %q", fields[0].Type)
				}
				if len(fields[0].Options) != 1 || fields[0].Options[0] != "draft" {
					t.Errorf("expected Options [\"draft\"], got %v", fields[0].Options)
				}
			},
		},
		{
			name:      "multiple fields with spaces",
			input:     "title, author, summary:paragraph",
			wantCount: 3,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Label != "title" {
					t.Errorf("field 0: expected Label \"title\", got %q", fields[0].Label)
				}
				if fields[1].Label != "author" {
					t.Errorf("field 1: expected Label \"author\", got %q", fields[1].Label)
				}
				if fields[2].Label != "summary" {
					t.Errorf("field 2: expected Label \"summary\", got %q", fields[2].Label)
				}
				if fields[2].Type != extractor.FieldTypeParagraph {
					t.Errorf("field 2: expected Type paragraph, got %q", fields[2].Type)
				}
			},
		},
		{
			name:      "empty string returns nil",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "number type",
			input:     "rating:number",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Type != extractor.FieldTypeNumber {
					t.Errorf("expected Type number, got %q", fields[0].Type)
				}
			},
		},
		{
			name:      "boolean type",
			input:     "is_draft:boolean",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Type != extractor.FieldTypeBoolean {
					t.Errorf("expected Type boolean, got %q", fields[0].Type)
				}
			},
		},
		{
			name:      "date type",
			input:     "published:date",
			wantCount: 1,
			check: func(t *testing.T, fields []extractor.Field) {
				if fields[0].Type != extractor.FieldTypeDate {
					t.Errorf("expected Type date, got %q", fields[0].Type)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fields, err := ParseFields(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(fields) != tc.wantCount {
				t.Fatalf("expected %d fields, got %d", tc.wantCount, len(fields))
			}
			if tc.check != nil {
				tc.check(t, fields)
			}
		})
	}
}

func TestParseFields_EnumWithMultipleOptions(t *testing.T) {
	// ParseFields splits on comma, which conflicts with enum options.
	// Test parseOneField directly for enum with multiple options.
	field, err := parseOneField("sentiment:enum[positive,negative,neutral]")
	if err != nil {
		t.Fatalf("parseOneField returned error: %v", err)
	}
	if field.Type != extractor.FieldTypeEnum {
		t.Errorf("expected Type enum, got %q", field.Type)
	}
	wantOpts := []string{"positive", "negative", "neutral"}
	if len(field.Options) != len(wantOpts) {
		t.Fatalf("expected %d options, got %d", len(wantOpts), len(field.Options))
	}
	for i, opt := range wantOpts {
		if field.Options[i] != opt {
			t.Errorf("option %d: expected %q, got %q", i, opt, field.Options[i])
		}
	}
}

func TestFormatMarkdown(t *testing.T) {
	fields := []extractor.Field{
		{Label: "title", Type: extractor.FieldTypeText},
		{Label: "tags", Type: extractor.FieldTypeList},
		{Label: "rating", Type: extractor.FieldTypeNumber},
	}

	result := &extractor.ExtractionResult{
		URL:         "https://example.com/article",
		ContentType: "article",
		ExtractedAt: time.Now(),
		Data: map[string]any{
			"title":  "Test Article",
			"tags":   []any{"go", "testing", "cli"},
			"rating": 4.5,
		},
		Model:      "claude-sonnet-4",
		TokensUsed: 500,
	}

	output := formatMarkdown(result, fields)

	// Verify URL appears in output.
	if !strings.Contains(output, "https://example.com/article") {
		t.Error("markdown output should contain the URL")
	}

	// Verify ## headers for each field.
	for _, f := range fields {
		header := "## " + f.Label
		if !strings.Contains(output, header) {
			t.Errorf("markdown output should contain header %q", header)
		}
	}

	// Verify list fields are rendered as bullet points.
	if !strings.Contains(output, "- go") {
		t.Error("markdown output should contain bullet point '- go'")
	}
	if !strings.Contains(output, "- testing") {
		t.Error("markdown output should contain bullet point '- testing'")
	}
	if !strings.Contains(output, "- cli") {
		t.Error("markdown output should contain bullet point '- cli'")
	}
}

func TestFormatCSV(t *testing.T) {
	fields := []extractor.Field{
		{Label: "title", Type: extractor.FieldTypeText},
		{Label: "tags", Type: extractor.FieldTypeList},
		{Label: "is_published", Type: extractor.FieldTypeBoolean},
	}

	result := &extractor.ExtractionResult{
		URL:         "https://example.com/article",
		ContentType: "article",
		ExtractedAt: time.Now(),
		Data: map[string]any{
			"title":        "Test Article",
			"tags":         []any{"go", "testing"},
			"is_published": true,
		},
		Model:      "claude-sonnet-4",
		TokensUsed: 500,
	}

	output := formatCSV(result, fields)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify exactly 2 lines (header + data).
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Verify header row has field labels.
	header := lines[0]
	for _, f := range fields {
		if !strings.Contains(header, f.Label) {
			t.Errorf("header should contain %q", f.Label)
		}
	}

	// Verify lists are joined with " | ".
	dataLine := lines[1]
	if !strings.Contains(dataLine, "go | testing") {
		t.Errorf("data line should contain list joined with ' | ', got %q", dataLine)
	}
}
