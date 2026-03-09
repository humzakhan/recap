package template

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/humzakhan/recap/internal/extractor"
)

//go:embed builtin/*.json
var builtinFS embed.FS

// LoadBuiltins reads all embedded JSON template files from the builtin/
// directory, parses each as an extractor.Template, and returns them sorted
// alphabetically by name.
func LoadBuiltins() ([]extractor.Template, error) {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return nil, fmt.Errorf("reading embedded builtin templates: %w", err)
	}

	var templates []extractor.Template
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := builtinFS.ReadFile("builtin/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading embedded template %s: %w", entry.Name(), err)
		}

		var tmpl extractor.Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			return nil, fmt.Errorf("parsing embedded template %s: %w", entry.Name(), err)
		}

		templates = append(templates, tmpl)
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}
