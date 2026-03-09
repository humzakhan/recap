package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
	"github.com/humzakhan/recap/internal/extractor"
)

const maxMediaFileSize = 100 * 1024 * 1024 // 100MB
const maxTextFileSize = 20 * 1024 * 1024   // 20MB

// mediaExtensions contains file extensions considered media (video/audio).
var mediaExtensions = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".webm": true,
	".avi":  true,
	".mp3":  true,
	".wav":  true,
	".m4a":  true,
	".ogg":  true,
	".flac": true,
}

// fetchLocal reads a local file and returns its content as FetchedContent.
func fetchLocal(ctx context.Context, rawPath string) (*extractor.FetchedContent, error) {
	resolved, err := resolvePath(rawPath)
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", rawPath, err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", resolved)
		}
		return nil, fmt.Errorf("stat %s: %w", resolved, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", resolved)
	}

	ext := strings.ToLower(filepath.Ext(resolved))
	size := info.Size()

	if mediaExtensions[ext] {
		if size > maxMediaFileSize {
			return nil, fmt.Errorf("file size %d bytes exceeds maximum %d bytes (100MB) for media files", size, maxMediaFileSize)
		}
	} else {
		if size > maxTextFileSize {
			return nil, fmt.Errorf("file size %d bytes exceeds maximum %d bytes (20MB) for text files", size, maxTextFileSize)
		}
	}

	switch ext {
	case ".pdf":
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", resolved, err)
		}
		return fetchPDFFromReader(data, rawPath)

	case ".mp4", ".mov", ".webm", ".avi":
		return nil, fmt.Errorf("local video transcription not yet implemented")

	case ".mp3", ".wav", ".m4a", ".ogg", ".flac":
		return nil, fmt.Errorf("local audio transcription not yet implemented")

	case ".html", ".htm":
		return fetchLocalHTML(resolved, rawPath)

	default:
		return fetchLocalText(resolved, rawPath, ext)
	}
}

// fetchLocalHTML reads an HTML file and extracts article content using readability.
func fetchLocalHTML(resolved, rawPath string) (*extractor.FetchedContent, error) {
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", resolved, err)
	}

	sourceURL, err := url.Parse("file://" + resolved)
	if err != nil {
		return nil, fmt.Errorf("constructing source URL for %s: %w", resolved, err)
	}

	article, err := readability.FromReader(bytes.NewReader(data), sourceURL)
	if err != nil {
		return nil, fmt.Errorf("extracting content from HTML file %s: %w", resolved, err)
	}

	title := article.Title
	if title == "" {
		title = filenameWithoutExt(resolved)
	}

	return &extractor.FetchedContent{
		URL:         rawPath,
		ContentType: "article",
		Title:       title,
		RawText:     article.TextContent,
		FetchedAt:   time.Now(),
	}, nil
}

// fetchLocalText reads a text file and returns its content directly.
func fetchLocalText(resolved, rawPath, ext string) (*extractor.FetchedContent, error) {
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", resolved, err)
	}

	return &extractor.FetchedContent{
		URL:         rawPath,
		ContentType: inferContentType(ext),
		Title:       filenameWithoutExt(resolved),
		RawText:     string(data),
		FetchedAt:   time.Now(),
	}, nil
}

// resolvePath expands ~, strips file:// scheme, and resolves relative paths.
func resolvePath(rawPath string) (string, error) {
	path := rawPath

	// Strip file:// scheme.
	if strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}

	// Expand ~ to home directory.
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Resolve relative paths against the working directory.
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
		path = filepath.Join(wd, path)
	}

	return filepath.Clean(path), nil
}

// inferContentType maps a file extension to a content type string.
func inferContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf":
		return "pdf"
	case ".mp4", ".mov", ".webm", ".avi":
		return "video"
	case ".mp3", ".wav", ".m4a", ".ogg", ".flac":
		return "audio"
	case ".html", ".htm":
		return "article"
	default:
		return "text"
	}
}

// filenameWithoutExt returns the base filename without its extension.
func filenameWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
