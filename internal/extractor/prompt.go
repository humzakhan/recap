package extractor

import "encoding/json"

// DefaultMaxContentChars is the default maximum character count for content
// sent to the LLM. Content exceeding this limit is truncated.
const DefaultMaxContentChars = 80000

// SystemPrompt is the canonical system prompt used for structured extraction.
const SystemPrompt = `You are a structured data extraction assistant. You will be given the text content
of a web page and a list of fields to extract. Your job is to extract the requested
information and return it as a single, flat JSON object.

Rules:
- Return ONLY a valid JSON object. No preamble, no markdown fences, no explanation.
- Every field in the schema must appear as a key in the output, even if the value is null.
- Keys must be the field labels converted to snake_case.
- Respect the type constraints for each field:
  - text / paragraph: a string
  - list: an array of strings
  - enum: exactly one of the provided options as a string
  - number: a numeric value (no units, no quotes)
  - boolean: true or false
  - date: an ISO 8601 date string, or null if not found
- If a field cannot be determined from the content, return null for that key.
- Do not infer or fabricate information that is not present in the content.`

// promptField is an internal type used to serialize fields into the user prompt.
// It uses snake_case labels and includes options for enum fields.
type promptField struct {
	Label   string   `json:"label"`
	Type    string   `json:"type"`
	Options []string `json:"options,omitempty"`
}

// BuildPrompt constructs the system and user prompts for the extraction LLM call.
// It converts field labels to snake_case and truncates content to DefaultMaxContentChars.
func BuildPrompt(content FetchedContent, fields []Field) (systemPrompt string, userPrompt string) {
	systemPrompt = SystemPrompt

	// Build the prompt-ready field list with snake_case labels.
	pFields := make([]promptField, len(fields))
	for i, f := range fields {
		pFields[i] = promptField{
			Label:   ToSnakeCase(f.Label),
			Type:    string(f.Type),
			Options: f.Options,
		}
	}

	fieldsJSON, _ := json.MarshalIndent(pFields, "", "  ")

	truncated := TruncateContent(content.RawText, DefaultMaxContentChars)

	userPrompt = "Schema:\n" + string(fieldsJSON) + "\n\nContent:\n" + truncated
	return systemPrompt, userPrompt
}

// TruncateContent truncates text to maxChars if it exceeds that length.
// It keeps the first 60% from the beginning and the last 40% from the end,
// inserting a truncation marker between them.
func TruncateContent(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}

	const marker = "\n\n[...content truncated...]\n\n"

	// Reserve space for the marker so total output stays within maxChars.
	available := maxChars - len(marker)
	if available < 0 {
		available = 0
	}

	headSize := available * 60 / 100
	tailSize := available - headSize

	return text[:headSize] + marker + text[len(text)-tailSize:]
}
