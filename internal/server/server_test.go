package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/humzakhan/recap/internal/config"
)

func testServer() *Server {
	cfg := &config.Config{
		AnthropicAPIKey: "test-key",
		Model:           "test-model",
		MaxTokens:       100,
		DefaultFormat:   "json",
		Server:          config.ServerConfig{Port: 0, Host: "127.0.0.1"},
	}
	return New(cfg)
}

func TestHealthEndpoint(t *testing.T) {
	s := testServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("expected body to contain '\"status\":\"ok\"', got %s", body)
	}
}

func TestListTemplates(t *testing.T) {
	s := testServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var templates []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
		t.Fatalf("failed to parse response as JSON array: %v", err)
	}

	if len(templates) < 5 {
		t.Fatalf("expected at least 5 built-in templates, got %d", len(templates))
	}
}

func TestGetTemplate(t *testing.T) {
	s := testServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/news-article", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var tmpl struct {
		Name   string            `json:"name"`
		Fields []json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tmpl); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if tmpl.Name != "news-article" {
		t.Fatalf("expected name 'news-article', got %q", tmpl.Name)
	}

	if len(tmpl.Fields) == 0 {
		t.Fatal("expected non-empty fields array")
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	s := testServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/nonexistent", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Code != "TEMPLATE_NOT_FOUND" {
		t.Fatalf("expected code 'TEMPLATE_NOT_FOUND', got %q", resp.Code)
	}
}

func TestExtract_NoFields(t *testing.T) {
	s := testServer()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Code != "INVALID_REQUEST" {
		t.Fatalf("expected code 'INVALID_REQUEST', got %q", resp.Code)
	}

	if !strings.Contains(strings.ToLower(resp.Error), "no fields") {
		t.Fatalf("expected error to mention 'no fields', got %q", resp.Error)
	}
}

func TestExtract_InvalidBody(t *testing.T) {
	s := testServer()

	body := `{this is not valid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Code != "INVALID_REQUEST" {
		t.Fatalf("expected code 'INVALID_REQUEST', got %q", resp.Code)
	}
}

func TestExtract_WithTemplate(t *testing.T) {
	s := testServer()

	body := `{"url":"https://example.com","template":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Code != "TEMPLATE_NOT_FOUND" {
		t.Fatalf("expected code 'TEMPLATE_NOT_FOUND', got %q", resp.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimiter(2))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// First two requests should succeed.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate-limited.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 3: expected status 429, got %d", w.Code)
	}
}
