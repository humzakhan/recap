package template

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/humzakhan/recap/internal/extractor"
)

// LoadTemplate loads a template by name or file path. It checks in order:
//  1. If nameOrPath is a file path that exists on disk, load it directly.
//  2. User templates in ~/.recap/templates/.
//  3. Built-in embedded templates.
func LoadTemplate(nameOrPath string) (*extractor.Template, error) {
	// 1. Check if it's a file path.
	if _, err := os.Stat(nameOrPath); err == nil {
		data, err := os.ReadFile(nameOrPath)
		if err != nil {
			return nil, fmt.Errorf("reading template file %s: %w", nameOrPath, err)
		}
		var tmpl extractor.Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			return nil, fmt.Errorf("parsing template file %s: %w", nameOrPath, err)
		}
		return &tmpl, nil
	}

	// 2. Check user templates.
	userTmpl, err := LoadUserTemplate(nameOrPath)
	if err == nil {
		return userTmpl, nil
	}

	// 3. Check builtins.
	builtins, err := LoadBuiltins()
	if err != nil {
		return nil, fmt.Errorf("loading builtin templates: %w", err)
	}
	for i := range builtins {
		if strings.EqualFold(builtins[i].Name, nameOrPath) {
			return &builtins[i], nil
		}
	}

	return nil, fmt.Errorf("template %q not found", nameOrPath)
}

// ListTemplates returns all available templates, combining builtins and
// user-defined templates. The result is sorted alphabetically by name.
// If a user template has the same name as a builtin, the user template
// takes precedence.
func ListTemplates() ([]extractor.Template, error) {
	builtins, err := LoadBuiltins()
	if err != nil {
		return nil, fmt.Errorf("loading builtin templates: %w", err)
	}

	userTemplates, err := ListUserTemplates()
	if err != nil {
		return nil, fmt.Errorf("loading user templates: %w", err)
	}

	// Index by name; user templates override builtins.
	byName := make(map[string]extractor.Template, len(builtins)+len(userTemplates))
	for _, t := range builtins {
		byName[strings.ToLower(t.Name)] = t
	}
	for _, t := range userTemplates {
		byName[strings.ToLower(t.Name)] = t
	}

	result := make([]extractor.Template, 0, len(byName))
	for _, t := range byName {
		result = append(result, t)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}
