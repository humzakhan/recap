package aiclient

// Provider constants.
const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGoogle    = "google"
)

// ModelInfo describes a known AI model.
type ModelInfo struct {
	ID       string `json:"id"`       // API model identifier
	Name     string `json:"name"`     // Human-friendly display name
	Provider string `json:"provider"` // Provider key
}

// KnownModels is the registry of well-known models that recap supports.
// Users can use any model string, but these are the ones we list in
// `recap config models` and use for provider auto-detection.
var KnownModels = []ModelInfo{
	// Anthropic
	{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Provider: ProviderAnthropic},
	{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", Provider: ProviderAnthropic},
	{ID: "claude-haiku-4-20250514", Name: "Claude Haiku 4", Provider: ProviderAnthropic},

	// OpenAI
	{ID: "gpt-4o", Name: "GPT-4o", Provider: ProviderOpenAI},
	{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: ProviderOpenAI},
	{ID: "o3", Name: "O3", Provider: ProviderOpenAI},
	{ID: "o3-mini", Name: "O3 Mini", Provider: ProviderOpenAI},
	{ID: "o4-mini", Name: "O4 Mini", Provider: ProviderOpenAI},

	// Google
	{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Provider: ProviderGoogle},
	{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Provider: ProviderGoogle},
	{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Provider: ProviderGoogle},
}

// LookupModel finds a model in the registry by its ID.
func LookupModel(modelID string) (ModelInfo, bool) {
	for _, m := range KnownModels {
		if m.ID == modelID {
			return m, true
		}
	}
	return ModelInfo{}, false
}

// ModelsByProvider returns all known models grouped by provider.
func ModelsByProvider() map[string][]ModelInfo {
	grouped := make(map[string][]ModelInfo)
	for _, m := range KnownModels {
		grouped[m.Provider] = append(grouped[m.Provider], m)
	}
	return grouped
}
