package extractor

import (
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"author", "author"},
		{"Key Takeaways", "key_takeaways"},
		{"author-name", "author_name"},
		{"URL", "url"},
		{"keyTakeaways", "key_takeaways"},
		{"published_at", "published_at"},
		{"My.Field.Name", "my_field_name"},
		{"  spaces  ", "spaces"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateFieldType(t *testing.T) {
	validTypes := []FieldType{
		FieldTypeText,
		FieldTypeList,
		FieldTypeEnum,
		FieldTypeNumber,
		FieldTypeBoolean,
		FieldTypeDate,
		FieldTypeParagraph,
	}

	for _, ft := range validTypes {
		t.Run(string(ft), func(t *testing.T) {
			if !ValidateFieldType(ft) {
				t.Errorf("ValidateFieldType(%q) = false, want true", ft)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		if ValidateFieldType("invalid") {
			t.Error("ValidateFieldType(\"invalid\") = true, want false")
		}
	})
}

func TestCoerceValue(t *testing.T) {
	t.Run("nil value returns nil for all types", func(t *testing.T) {
		types := []FieldType{
			FieldTypeText, FieldTypeList, FieldTypeEnum,
			FieldTypeNumber, FieldTypeBoolean, FieldTypeDate, FieldTypeParagraph,
		}
		for _, ft := range types {
			got, err := CoerceValue(nil, ft, nil)
			if err != nil {
				t.Errorf("CoerceValue(nil, %q): unexpected error: %v", ft, err)
			}
			if got != nil {
				t.Errorf("CoerceValue(nil, %q) = %v, want nil", ft, got)
			}
		}
	})

	t.Run("text", func(t *testing.T) {
		tests := []struct {
			name  string
			input any
			want  string
		}{
			{"string passthrough", "hello", "hello"},
			{"number to string", float64(42), "42"},
			{"bool to string", true, "true"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := CoerceValue(tt.input, FieldTypeText, nil)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				s, ok := got.(string)
				if !ok {
					t.Fatalf("expected string, got %T", got)
				}
				if s != tt.want {
					t.Errorf("got %q, want %q", s, tt.want)
				}
			})
		}
	})

	t.Run("list", func(t *testing.T) {
		t.Run("slice passthrough", func(t *testing.T) {
			input := []any{"a", "b", "c"}
			got, err := CoerceValue(input, FieldTypeList, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			arr, ok := got.([]any)
			if !ok {
				t.Fatalf("expected []any, got %T", got)
			}
			if len(arr) != 3 {
				t.Errorf("got length %d, want 3", len(arr))
			}
		})

		t.Run("string wraps to single-element array", func(t *testing.T) {
			got, err := CoerceValue("single", FieldTypeList, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			arr, ok := got.([]any)
			if !ok {
				t.Fatalf("expected []any, got %T", got)
			}
			if len(arr) != 1 || arr[0] != "single" {
				t.Errorf("got %v, want [single]", arr)
			}
		})
	})

	t.Run("enum", func(t *testing.T) {
		options := []string{"positive", "negative", "neutral"}

		t.Run("valid option", func(t *testing.T) {
			got, err := CoerceValue("positive", FieldTypeEnum, options)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "positive" {
				t.Errorf("got %v, want %q", got, "positive")
			}
		})

		t.Run("invalid option returns nil", func(t *testing.T) {
			got, err := CoerceValue("unknown", FieldTypeEnum, options)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != nil {
				t.Errorf("got %v, want nil", got)
			}
		})

		t.Run("case-insensitive match returns canonical option", func(t *testing.T) {
			got, err := CoerceValue("POSITIVE", FieldTypeEnum, options)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "positive" {
				t.Errorf("got %v, want %q", got, "positive")
			}
		})
	})

	t.Run("number", func(t *testing.T) {
		t.Run("float64 passthrough", func(t *testing.T) {
			got, err := CoerceValue(float64(3.14), FieldTypeNumber, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != float64(3.14) {
				t.Errorf("got %v, want 3.14", got)
			}
		})

		t.Run("string parses", func(t *testing.T) {
			got, err := CoerceValue("42.5", FieldTypeNumber, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != float64(42.5) {
				t.Errorf("got %v, want 42.5", got)
			}
		})

		t.Run("non-numeric string returns nil", func(t *testing.T) {
			got, err := CoerceValue("abc", FieldTypeNumber, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != nil {
				t.Errorf("got %v, want nil", got)
			}
		})
	})

	t.Run("boolean", func(t *testing.T) {
		tests := []struct {
			name string
			input any
			want  any
		}{
			{"true passthrough", true, true},
			{"false passthrough", false, false},
			{"yes to true", "yes", true},
			{"no to false", "no", false},
			{"1 to true", "1", true},
			{"0 to false", "0", false},
			{"maybe returns nil", "maybe", nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := CoerceValue(tt.input, FieldTypeBoolean, nil)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("got %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("date", func(t *testing.T) {
		t.Run("string passthrough", func(t *testing.T) {
			got, err := CoerceValue("2024-01-15", FieldTypeDate, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "2024-01-15" {
				t.Errorf("got %v, want %q", got, "2024-01-15")
			}
		})

		t.Run("non-string returns nil", func(t *testing.T) {
			got, err := CoerceValue(float64(12345), FieldTypeDate, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != nil {
				t.Errorf("got %v, want nil", got)
			}
		})
	})
}

func TestMergeFields(t *testing.T) {
	t.Run("empty base returns override", func(t *testing.T) {
		override := []Field{
			{Label: "title", Type: FieldTypeText},
		}
		got := MergeFields(nil, override)
		if len(got) != 1 || got[0].Label != "title" {
			t.Errorf("got %v, want %v", got, override)
		}
	})

	t.Run("base with empty override returns base", func(t *testing.T) {
		base := []Field{
			{Label: "title", Type: FieldTypeText},
			{Label: "author", Type: FieldTypeText},
		}
		got := MergeFields(base, nil)
		if len(got) != 2 {
			t.Errorf("got length %d, want 2", len(got))
		}
	})

	t.Run("override replaces matching label case-insensitive", func(t *testing.T) {
		base := []Field{
			{Label: "title", Type: FieldTypeText},
		}
		override := []Field{
			{Label: "Title", Type: FieldTypeParagraph},
		}
		got := MergeFields(base, override)
		if len(got) != 1 {
			t.Fatalf("got length %d, want 1", len(got))
		}
		if got[0].Type != FieldTypeParagraph {
			t.Errorf("got type %q, want %q", got[0].Type, FieldTypeParagraph)
		}
	})

	t.Run("override with new label appends", func(t *testing.T) {
		base := []Field{
			{Label: "title", Type: FieldTypeText},
		}
		override := []Field{
			{Label: "author", Type: FieldTypeText},
		}
		got := MergeFields(base, override)
		if len(got) != 2 {
			t.Fatalf("got length %d, want 2", len(got))
		}
		if got[0].Label != "title" || got[1].Label != "author" {
			t.Errorf("got labels [%q, %q], want [title, author]", got[0].Label, got[1].Label)
		}
	})

	t.Run("mixed replace and append", func(t *testing.T) {
		base := []Field{
			{Label: "title", Type: FieldTypeText},
			{Label: "author", Type: FieldTypeText},
		}
		override := []Field{
			{Label: "Author", Type: FieldTypeParagraph},  // replaces
			{Label: "summary", Type: FieldTypeParagraph}, // appends
		}
		got := MergeFields(base, override)
		if len(got) != 3 {
			t.Fatalf("got length %d, want 3", len(got))
		}
		// title stays at index 0
		if got[0].Label != "title" {
			t.Errorf("index 0: got label %q, want %q", got[0].Label, "title")
		}
		// author replaced in-place at index 1
		if got[1].Label != "Author" || got[1].Type != FieldTypeParagraph {
			t.Errorf("index 1: got {%q, %q}, want {Author, paragraph}", got[1].Label, got[1].Type)
		}
		// summary appended at index 2
		if got[2].Label != "summary" {
			t.Errorf("index 2: got label %q, want %q", got[2].Label, "summary")
		}
	})
}
