package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/humzakhan/recap/internal/extractor"
)

// ParseFields parses a comma-separated field definition string into a slice of Fields.
// Supported formats:
//   - "author"                              -> Field{Label:"author", Type:FieldTypeText}
//   - "author:text"                         -> Field{Label:"author", Type:FieldTypeText}
//   - "takeaways:list"                      -> Field{Label:"takeaways", Type:FieldTypeList}
//   - "sentiment:enum[positive,negative]"   -> Field{Label:"sentiment", Type:FieldTypeEnum, Options:["positive","negative"]}
func ParseFields(input string) ([]extractor.Field, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	var fields []extractor.Field
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		field, err := parseOneField(part)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	return fields, nil
}

// parseOneField parses a single field definition like "label:type[opt1,opt2]".
func parseOneField(s string) (extractor.Field, error) {
	// Check for enum options: label:enum[opt1,opt2,opt3]
	if idx := strings.Index(s, "["); idx != -1 {
		// Must end with ]
		if !strings.HasSuffix(s, "]") {
			return extractor.Field{}, fmt.Errorf("invalid field definition %q: unclosed bracket", s)
		}
		optionsStr := s[idx+1 : len(s)-1]
		beforeBracket := s[:idx]

		// beforeBracket should be "label:type"
		colonIdx := strings.Index(beforeBracket, ":")
		if colonIdx == -1 {
			return extractor.Field{}, fmt.Errorf("invalid field definition %q: options require a type (e.g., label:enum[...])", s)
		}
		label := strings.TrimSpace(beforeBracket[:colonIdx])
		typStr := strings.TrimSpace(beforeBracket[colonIdx+1:])
		ft := extractor.FieldType(typStr)

		if !extractor.ValidateFieldType(ft) {
			return extractor.Field{}, fmt.Errorf("invalid field type %q in %q", typStr, s)
		}

		var options []string
		for _, opt := range strings.Split(optionsStr, ",") {
			opt = strings.TrimSpace(opt)
			if opt != "" {
				options = append(options, opt)
			}
		}

		return extractor.Field{Label: label, Type: ft, Options: options}, nil
	}

	// No brackets: "label" or "label:type"
	colonIdx := strings.Index(s, ":")
	if colonIdx == -1 {
		// Bare label — defaults to text.
		return extractor.Field{Label: s, Type: extractor.FieldTypeText}, nil
	}

	label := strings.TrimSpace(s[:colonIdx])
	typStr := strings.TrimSpace(s[colonIdx+1:])

	if label == "" {
		return extractor.Field{}, fmt.Errorf("invalid field definition %q: empty label", s)
	}

	ft := extractor.FieldType(typStr)
	if !extractor.ValidateFieldType(ft) {
		return extractor.Field{}, fmt.Errorf("invalid field type %q in %q", typStr, s)
	}

	return extractor.Field{Label: label, Type: ft}, nil
}

// formatJSON pretty-prints the extraction result as JSON.
func formatJSON(result *extractor.ExtractionResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}
	return string(data), nil
}

// formatMarkdown renders the extraction result as a Markdown document.
func formatMarkdown(result *extractor.ExtractionResult, fields []extractor.Field) string {
	var sb strings.Builder
	sb.WriteString("# ")
	sb.WriteString(result.URL)
	sb.WriteString("\n\n")

	for _, f := range fields {
		key := extractor.ToSnakeCase(f.Label)
		val, ok := result.Data[key]
		if !ok {
			val = nil
		}

		sb.WriteString("## ")
		sb.WriteString(f.Label)
		sb.WriteString("\n\n")

		switch v := val.(type) {
		case nil:
			sb.WriteString("_Not available_\n")
		case []any:
			for _, item := range v {
				sb.WriteString("- ")
				sb.WriteString(fmt.Sprintf("%v", item))
				sb.WriteString("\n")
			}
		case bool:
			if v {
				sb.WriteString("Yes\n")
			} else {
				sb.WriteString("No\n")
			}
		default:
			sb.WriteString(fmt.Sprintf("%v\n", v))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatCSV renders the extraction result as a CSV row with a header.
func formatCSV(result *extractor.ExtractionResult, fields []extractor.Field) string {
	var headers []string
	var values []string

	for _, f := range fields {
		key := extractor.ToSnakeCase(f.Label)
		headers = append(headers, f.Label)

		val, ok := result.Data[key]
		if !ok || val == nil {
			values = append(values, "")
			continue
		}

		switch v := val.(type) {
		case []any:
			var items []string
			for _, item := range v {
				items = append(items, fmt.Sprintf("%v", item))
			}
			values = append(values, csvEscape(strings.Join(items, " | ")))
		case bool:
			if v {
				values = append(values, "true")
			} else {
				values = append(values, "false")
			}
		case string:
			values = append(values, csvEscape(v))
		default:
			values = append(values, csvEscape(fmt.Sprintf("%v", v)))
		}
	}

	return strings.Join(headers, ",") + "\n" + strings.Join(values, ",") + "\n"
}

// csvEscape wraps a value in double quotes if it contains commas, quotes, or newlines.
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
