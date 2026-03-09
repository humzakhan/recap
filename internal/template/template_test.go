package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/humzakhan/recap/internal/extractor"
)

func TestLoadBuiltins(t *testing.T) {
	templates, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins() returned error: %v", err)
	}

	if len(templates) != 5 {
		t.Fatalf("expected 5 builtin templates, got %d", len(templates))
	}

	for i, tmpl := range templates {
		if tmpl.Name == "" {
			t.Errorf("template %d has empty Name", i)
		}
		if len(tmpl.Fields) == 0 {
			t.Errorf("template %q has no fields", tmpl.Name)
		}
	}

	// Verify "news-article" template exists and has a "headline" field.
	var found bool
	for _, tmpl := range templates {
		if tmpl.Name == "news-article" {
			found = true
			var hasHeadline bool
			for _, f := range tmpl.Fields {
				if f.Label == "headline" {
					hasHeadline = true
					break
				}
			}
			if !hasHeadline {
				t.Error("news-article template missing 'headline' field")
			}
			break
		}
	}
	if !found {
		t.Error("news-article template not found in builtins")
	}
}

func TestLoadTemplate_Builtin(t *testing.T) {
	tmpl, err := LoadTemplate("news-article")
	if err != nil {
		t.Fatalf("LoadTemplate(\"news-article\") returned error: %v", err)
	}

	if tmpl.Name != "news-article" {
		t.Errorf("expected Name \"news-article\", got %q", tmpl.Name)
	}

	if len(tmpl.Fields) == 0 {
		t.Fatal("expected non-empty Fields for news-article")
	}

	// Check for expected fields.
	expectedLabels := map[string]bool{
		"headline":     false,
		"author":       false,
		"publication":  false,
		"published_at": false,
		"summary":      false,
		"key_points":   false,
		"sentiment":    false,
		"topics":       false,
	}
	for _, f := range tmpl.Fields {
		if _, ok := expectedLabels[f.Label]; ok {
			expectedLabels[f.Label] = true
		}
	}
	for label, found := range expectedLabels {
		if !found {
			t.Errorf("expected field %q not found in news-article template", label)
		}
	}
}

func TestLoadTemplate_NotFound(t *testing.T) {
	_, err := LoadTemplate("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent template, got nil")
	}
}

func TestSaveAndLoadUserTemplate(t *testing.T) {
	// Create a temporary directory to act as a template file on disk.
	// Since SaveUserTemplate/LoadUserTemplate use os.UserHomeDir() which
	// cannot be overridden easily, we test using file-path-based LoadTemplate
	// and direct file I/O instead.
	tmpDir := t.TempDir()

	tmpl := extractor.Template{
		Name:        "test-template",
		Description: "A test template",
		Fields: []extractor.Field{
			{Label: "title", Type: extractor.FieldTypeText},
			{Label: "tags", Type: extractor.FieldTypeList},
			{Label: "rating", Type: extractor.FieldTypeNumber},
		},
	}

	// Save template to temp dir as JSON.
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal template: %v", err)
	}

	filePath := filepath.Join(tmpDir, "test-template.json")
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Load it back via file path.
	loaded, err := LoadTemplate(filePath)
	if err != nil {
		t.Fatalf("LoadTemplate(%q) returned error: %v", filePath, err)
	}

	if loaded.Name != tmpl.Name {
		t.Errorf("expected Name %q, got %q", tmpl.Name, loaded.Name)
	}
	if loaded.Description != tmpl.Description {
		t.Errorf("expected Description %q, got %q", tmpl.Description, loaded.Description)
	}
	if len(loaded.Fields) != len(tmpl.Fields) {
		t.Fatalf("expected %d fields, got %d", len(tmpl.Fields), len(loaded.Fields))
	}
	for i, f := range loaded.Fields {
		if f.Label != tmpl.Fields[i].Label {
			t.Errorf("field %d: expected Label %q, got %q", i, tmpl.Fields[i].Label, f.Label)
		}
		if f.Type != tmpl.Fields[i].Type {
			t.Errorf("field %d: expected Type %q, got %q", i, tmpl.Fields[i].Type, f.Type)
		}
	}

	// Delete the file.
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("failed to remove template file: %v", err)
	}

	// Verify load fails after deletion.
	_, err = LoadTemplate(filePath)
	if err == nil {
		t.Error("expected error loading deleted template file, got nil")
	}
}

func TestListTemplates(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates() returned error: %v", err)
	}

	if len(templates) < 5 {
		t.Errorf("expected at least 5 templates (builtins), got %d", len(templates))
	}
}
