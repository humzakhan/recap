package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/humzakhan/recap/internal/extractor"
)

// userTemplateDir returns the path to ~/.recap/templates/, creating it if needed.
func userTemplateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".recap", "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create templates directory: %w", err)
	}
	return dir, nil
}

// templateFileName converts a template name to a filename.
func templateFileName(name string) string {
	return name + ".json"
}

// loadFromFile reads and parses a template JSON file.
func loadFromFile(path string) (*extractor.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tmpl extractor.Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parsing template file %s: %w", path, err)
	}
	return &tmpl, nil
}

// LoadUserTemplate loads a single user template by name from ~/.recap/templates/.
func LoadUserTemplate(name string) (*extractor.Template, error) {
	dir, err := userTemplateDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, templateFileName(name))
	return loadFromFile(path)
}

// ListUserTemplates returns all user-defined templates from ~/.recap/templates/.
func ListUserTemplates() ([]extractor.Template, error) {
	dir, err := userTemplateDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading user templates directory: %w", err)
	}

	var templates []extractor.Template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		tmpl, err := loadFromFile(filepath.Join(dir, entry.Name()))
		if err == nil {
			templates = append(templates, *tmpl)
		}
	}
	return templates, nil
}

// SaveUserTemplate saves a template to the user template directory.
func SaveUserTemplate(tmpl *extractor.Template) error {
	dir, err := userTemplateDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling template: %w", err)
	}

	path := filepath.Join(dir, templateFileName(tmpl.Name))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing template file: %w", err)
	}
	return nil
}

// DeleteUserTemplate removes a user template by name.
func DeleteUserTemplate(name string) error {
	dir, err := userTemplateDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, templateFileName(name))
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting template %q: %w", name, err)
	}
	return nil
}

// ImportTemplate reads a template from a file path and saves it as a user template.
func ImportTemplate(filePath string) (*extractor.Template, error) {
	tmpl, err := loadFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading template file: %w", err)
	}
	if tmpl.Name == "" {
		base := filepath.Base(filePath)
		tmpl.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if err := SaveUserTemplate(tmpl); err != nil {
		return nil, err
	}
	return tmpl, nil
}

// ExportTemplate writes a template to a file.
func ExportTemplate(name string, outputPath string) error {
	tmpl, err := LoadTemplate(name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling template: %w", err)
	}
	data = append(data, '\n')

	if outputPath == "" {
		outputPath = templateFileName(name)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("writing template file: %w", err)
	}
	return nil
}
