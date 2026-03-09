package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/absolute/path", true},
		{"./relative", true},
		{"~/home", true},
		{"file:///path", true},
		{"https://example.com", false},
		{"http://example.com", false},
		{"example.com", true}, // no scheme = local per spec
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isLocalPath(tt.input)
			if got != tt.want {
				t.Errorf("isLocalPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFetchHTTP(t *testing.T) {
	paragraph := strings.Repeat("This is a test paragraph with enough content to pass the readability minimum character threshold. ", 5)
	html := fmt.Sprintf(`<html><head><title>Test Page</title></head><body><article><p>%s</p></article></body></html>`, paragraph)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
	defer srv.Close()

	ctx := context.Background()
	result, err := fetchHTTP(ctx, srv.URL)
	if err != nil {
		t.Fatalf("fetchHTTP returned error: %v", err)
	}

	if result.ContentType != "article" {
		t.Errorf("ContentType = %q, want %q", result.ContentType, "article")
	}

	if result.Title == "" {
		t.Error("Title is empty, expected non-empty")
	}

	if result.RawText == "" {
		t.Error("RawText is empty, expected non-empty")
	}

	if !strings.Contains(result.RawText, "test paragraph") {
		t.Error("RawText does not contain paragraph content")
	}
}

func TestFetchHTTP_Fallback(t *testing.T) {
	html := `<html><body>Short</body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
	defer srv.Close()

	ctx := context.Background()
	result, err := fetchHTTP(ctx, srv.URL)
	if err != nil {
		t.Fatalf("fetchHTTP returned error: %v", err)
	}

	if result.RawText == "" {
		t.Error("RawText is empty, expected fallback content")
	}

	if !strings.Contains(result.RawText, "Short") {
		t.Errorf("RawText = %q, expected it to contain %q", result.RawText, "Short")
	}
}

func TestFetchHTTP_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	ctx := context.Background()
	_, err := fetchHTTP(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, expected it to contain %q", err.Error(), "404")
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple paragraph",
			input: "<p>hello</p>",
			want:  "hello",
		},
		{
			name:  "multiple tags",
			input: "<b>bold</b> and <i>italic</i>",
			want:  "bold and italic",
		},
		{
			name:  "no tags",
			input: "no tags",
			want:  "no tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTMLTags(tt.input)
			if got != tt.want {
				t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFetchLocal_TextFile(t *testing.T) {
	content := "Hello, this is test content for the local fetcher."

	tmpFile, err := os.CreateTemp(t.TempDir(), "testfile-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	ctx := context.Background()
	result, err := fetchLocal(ctx, tmpFile.Name())
	if err != nil {
		t.Fatalf("fetchLocal returned error: %v", err)
	}

	if result.ContentType != "text" {
		t.Errorf("ContentType = %q, want %q", result.ContentType, "text")
	}

	if result.RawText != content {
		t.Errorf("RawText = %q, want %q", result.RawText, content)
	}

	// Title should be the filename without extension.
	base := filepath.Base(tmpFile.Name())
	ext := filepath.Ext(base)
	expectedTitle := strings.TrimSuffix(base, ext)
	if result.Title != expectedTitle {
		t.Errorf("Title = %q, want %q", result.Title, expectedTitle)
	}
}

func TestFetchLocal_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := fetchLocal(ctx, "/nonexistent/path/to/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error = %q, expected it to contain %q", err.Error(), "file not found")
	}
}

func TestFetchLocal_Directory(t *testing.T) {
	dir := t.TempDir()

	ctx := context.Background()
	_, err := fetchLocal(ctx, dir)
	if err == nil {
		t.Fatal("expected error for directory, got nil")
	}

	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error = %q, expected it to contain %q", err.Error(), "directory")
	}
}

func TestRouteByExtension(t *testing.T) {
	tests := []struct {
		name    string
		urlPath string
		wantNil bool
	}{
		{"pdf extension", "http://example.com/doc.pdf", false},
		{"mp4 extension", "http://example.com/video.mp4", false},
		{"mp3 extension", "http://example.com/audio.mp3", false},
		{"html extension", "http://example.com/page.html", true},
		{"no extension", "http://example.com/page", true},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.urlPath)
			if err != nil {
				t.Fatalf("failed to parse URL %q: %v", tt.urlPath, err)
			}

			result := routeByExtension(ctx, tt.urlPath, parsed)
			if tt.wantNil && result != nil {
				t.Errorf("routeByExtension(%q) returned non-nil, want nil", tt.urlPath)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("routeByExtension(%q) returned nil, want non-nil", tt.urlPath)
			}
		})
	}
}

func TestInferContentType(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".pdf", "pdf"},
		{".mp4", "video"},
		{".mp3", "audio"},
		{".html", "article"},
		{".txt", "text"},
		{".md", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := inferContentType(tt.ext)
			if got != tt.want {
				t.Errorf("inferContentType(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}
