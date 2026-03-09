package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/humzakhan/recap/internal/aiclient"
	"github.com/humzakhan/recap/internal/extractor"
	"github.com/humzakhan/recap/internal/fetcher"
	"github.com/humzakhan/recap/internal/template"
)

// --------------------------------------------------------------------------
// Request / response types
// --------------------------------------------------------------------------

type extractRequest struct {
	URL      string            `json:"url" binding:"required"`
	Fields   []extractor.Field `json:"fields"`
	Template string            `json:"template,omitempty"`
	Format   string            `json:"format,omitempty"`
}

type extractResponse struct {
	URL         string         `json:"url"`
	ContentType string         `json:"content_type"`
	ExtractedAt string         `json:"extracted_at"`
	Data        map[string]any `json:"data"`
	RawContent  string         `json:"raw_content"`
	Model       string         `json:"model"`
	TokensUsed  int            `json:"tokens_used"`
	Formatted   string         `json:"formatted"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// --------------------------------------------------------------------------
// Fetcher adapter
// --------------------------------------------------------------------------

// fetcherAdapter wraps the fetcher.Fetch package function to satisfy
// the extractor.Fetcher interface.
type fetcherAdapter struct{}

func (f fetcherAdapter) Fetch(ctx context.Context, rawURL string) (*extractor.FetchedContent, error) {
	return fetcher.Fetch(ctx, rawURL)
}

// --------------------------------------------------------------------------
// Handlers
// --------------------------------------------------------------------------

// handleHealth returns a simple health-check response.
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleExtract performs a structured extraction on the given URL.
func (s *Server) handleExtract(c *gin.Context) {
	var req extractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Resolve fields: load template if specified, then merge.
	fields := req.Fields
	if req.Template != "" {
		tmpl, err := template.LoadTemplate(req.Template)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse{
				Error: fmt.Sprintf("template %q not found: %v", req.Template, err),
				Code:  "TEMPLATE_NOT_FOUND",
			})
			return
		}
		fields = extractor.MergeFields(tmpl.Fields, fields)
	}

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "no fields specified; provide fields or a template name",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Default format to json.
	format := req.Format
	if format == "" {
		format = "json"
	}

	// Build and run the extraction using the server's AI client.
	ext := extractor.NewExtractor(
		fetcherAdapter{},
		s.AIClient,
		s.Config.MaxTokens,
	)

	result, err := ext.Extract(c.Request.Context(), extractor.ExtractionRequest{
		URL:    req.URL,
		Fields: fields,
		Format: format,
	})
	if err != nil {
		code := "EXTRACT_FAILED"
		if strings.Contains(err.Error(), "fetch failed") {
			code = "FETCH_FAILED"
		}
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: err.Error(),
			Code:  code,
		})
		return
	}

	// Format the result.
	formatted, err := formatResult(result, fields, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: fmt.Sprintf("formatting result: %v", err),
			Code:  "EXTRACT_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, extractResponse{
		URL:         result.URL,
		ContentType: result.ContentType,
		ExtractedAt: result.ExtractedAt.Format("2006-01-02T15:04:05Z07:00"),
		Data:        result.Data,
		RawContent:  result.RawContent,
		Model:       result.Model,
		TokensUsed:  result.TokensUsed,
		Formatted:   formatted,
	})
}

// handleListTemplates returns all available templates (built-in + user).
func (s *Server) handleListTemplates(c *gin.Context) {
	templates, err := template.ListTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: fmt.Sprintf("listing templates: %v", err),
			Code:  "EXTRACT_FAILED",
		})
		return
	}
	c.JSON(http.StatusOK, templates)
}

// handleGetTemplate returns a single template by name.
func (s *Server) handleGetTemplate(c *gin.Context) {
	name := c.Param("name")
	tmpl, err := template.LoadTemplate(name)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{
			Error: fmt.Sprintf("template %q not found: %v", name, err),
			Code:  "TEMPLATE_NOT_FOUND",
		})
		return
	}
	c.JSON(http.StatusOK, tmpl)
}

// handleCreateTemplate saves a new user template.
func (s *Server) handleCreateTemplate(c *gin.Context) {
	var tmpl extractor.Template
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
			Code:  "INVALID_REQUEST",
		})
		return
	}

	if tmpl.Name == "" {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "template name is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	if len(tmpl.Fields) == 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "template must have at least one field",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	if err := template.SaveUserTemplate(&tmpl); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: fmt.Sprintf("saving template: %v", err),
			Code:  "EXTRACT_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, tmpl)
}

// handleDeleteTemplate removes a user template by name.
func (s *Server) handleDeleteTemplate(c *gin.Context) {
	name := c.Param("name")
	if err := template.DeleteUserTemplate(name); err != nil {
		c.JSON(http.StatusNotFound, errorResponse{
			Error: fmt.Sprintf("deleting template %q: %v", name, err),
			Code:  "TEMPLATE_NOT_FOUND",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("template %q deleted", name)})
}

// handleListModels returns all known models grouped by provider.
func (s *Server) handleListModels(c *gin.Context) {
	type modelResponse struct {
		Models  []aiclient.ModelInfo `json:"models"`
		Current string               `json:"current"`
	}
	c.JSON(http.StatusOK, modelResponse{
		Models:  aiclient.KnownModels,
		Current: s.Config.Model,
	})
}

// --------------------------------------------------------------------------
// Format helpers (duplicated from cmd/recap since that package is main)
// --------------------------------------------------------------------------

// formatResult renders the extraction result in the requested format.
func formatResult(result *extractor.ExtractionResult, fields []extractor.Field, format string) (string, error) {
	switch format {
	case "markdown":
		return formatResultMarkdown(result, fields), nil
	case "csv":
		return formatResultCSV(result, fields), nil
	default:
		return formatResultJSON(result)
	}
}

// formatResultJSON pretty-prints the extraction result as JSON.
func formatResultJSON(result *extractor.ExtractionResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}
	return string(data), nil
}

// formatResultMarkdown renders the extraction result as a Markdown document.
func formatResultMarkdown(result *extractor.ExtractionResult, fields []extractor.Field) string {
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

// formatResultCSV renders the extraction result as a CSV row with a header.
func formatResultCSV(result *extractor.ExtractionResult, fields []extractor.Field) string {
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
