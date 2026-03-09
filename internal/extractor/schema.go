package extractor

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// FieldType represents the expected output type of an extracted field.
type FieldType string

const (
	FieldTypeText      FieldType = "text"
	FieldTypeList      FieldType = "list"
	FieldTypeEnum      FieldType = "enum"
	FieldTypeNumber    FieldType = "number"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeDate      FieldType = "date"
	FieldTypeParagraph FieldType = "paragraph"
)

// Field defines a single extraction target.
type Field struct {
	Label   string    `json:"label"`
	Type    FieldType `json:"type"`
	Options []string  `json:"options,omitempty"`
}

// Template is a named, reusable set of fields.
type Template struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Fields      []Field `json:"fields"`
}

// FetchedContent is the result of fetching a URL.
type FetchedContent struct {
	URL         string    `json:"url"`
	ContentType string    `json:"content_type"`
	Title       string    `json:"title,omitempty"`
	RawText     string    `json:"raw_text"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// ExtractionRequest is the input to Extract().
type ExtractionRequest struct {
	URL    string  `json:"url"`
	Fields []Field `json:"fields"`
	Format string  `json:"format"`
}

// ExtractionResult is the output of Extract().
type ExtractionResult struct {
	URL         string         `json:"url"`
	ContentType string         `json:"content_type"`
	ExtractedAt time.Time      `json:"extracted_at"`
	Data        map[string]any `json:"data"`
	RawContent  string         `json:"raw_content"`
	Model       string         `json:"model"`
	TokensUsed  int            `json:"tokens_used"`
}

// ToSnakeCase converts a label string to snake_case.
// It handles camelCase, spaces, hyphens, dots, consecutive underscores,
// and trims leading/trailing underscores.
func ToSnakeCase(label string) string {
	// First, insert underscores before uppercase letters in camelCase.
	var buf strings.Builder
	for i, r := range label {
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(label[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				buf.WriteByte('_')
			}
		}
		buf.WriteRune(r)
	}
	s := buf.String()

	// Lowercase the whole string.
	s = strings.ToLower(s)

	// Replace spaces, hyphens, and dots with underscores.
	replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_")
	s = replacer.Replace(s)

	// Collapse consecutive underscores.
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	// Trim leading/trailing underscores.
	s = strings.Trim(s, "_")

	return s
}

// ValidateFieldType returns true if the given field type is one of the known constants.
func ValidateFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldTypeText, FieldTypeList, FieldTypeEnum, FieldTypeNumber,
		FieldTypeBoolean, FieldTypeDate, FieldTypeParagraph:
		return true
	}
	return false
}

// CoerceValue coerces a parsed JSON value to match the declared field type.
// It returns (nil, nil) when the input value is nil, which represents a JSON null.
func CoerceValue(value any, fieldType FieldType, options []string) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch fieldType {
	case FieldTypeText, FieldTypeParagraph:
		return coerceText(value)
	case FieldTypeList:
		return coerceList(value)
	case FieldTypeEnum:
		return coerceEnum(value, options)
	case FieldTypeNumber:
		return coerceNumber(value)
	case FieldTypeBoolean:
		return coerceBoolean(value)
	case FieldTypeDate:
		return coerceDate(value)
	default:
		return nil, fmt.Errorf("unknown field type: %s", fieldType)
	}
}

func coerceText(value any) (any, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func coerceList(value any) (any, error) {
	switch v := value.(type) {
	case []any:
		return v, nil
	case string:
		return []any{v}, nil
	default:
		return nil, nil
	}
}

func coerceEnum(value any, options []string) (any, error) {
	s, ok := value.(string)
	if !ok {
		return nil, nil
	}
	for _, opt := range options {
		if strings.EqualFold(s, opt) {
			return opt, nil
		}
	}
	// Value not in allowed options — return nil (log-worthy but not fatal).
	return nil, nil
}

func coerceNumber(value any) (any, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, nil
		}
		return f, nil
	default:
		return nil, nil
	}
}

func coerceBoolean(value any) (any, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(v) {
		case "true", "yes", "1":
			return true, nil
		case "false", "no", "0":
			return false, nil
		}
		return nil, nil
	default:
		return nil, nil
	}
}

func coerceDate(value any) (any, error) {
	s, ok := value.(string)
	if !ok {
		return nil, nil
	}
	return s, nil
}

// MergeFields merges two field lists. Base fields come first, then override fields
// are appended. If a label exists in both (case-insensitive via ToSnakeCase), the
// override version replaces the base entry in-place, preserving its position.
func MergeFields(base []Field, override []Field) []Field {
	// Build the result starting from a copy of base.
	result := make([]Field, len(base))
	copy(result, base)

	// Index base fields by snake_case label for quick lookup.
	indexByLabel := make(map[string]int, len(base))
	for i, f := range result {
		indexByLabel[ToSnakeCase(f.Label)] = i
	}

	for _, of := range override {
		key := ToSnakeCase(of.Label)
		if idx, exists := indexByLabel[key]; exists {
			// Replace in-place.
			result[idx] = of
		} else {
			// Append new field.
			indexByLabel[key] = len(result)
			result = append(result, of)
		}
	}

	return result
}
