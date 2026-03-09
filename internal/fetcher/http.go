package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
	"github.com/humzakhan/recap/internal/extractor"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	httpTimeout      = 30 * time.Second
	maxRedirects     = 5
	minReadableChars = 200
)

// fetchHTTP fetches a URL using a standard HTTP GET, extracts article content
// using go-readability, and falls back to HTML tag stripping if the extracted
// text is too short.
func fetchHTTP(ctx context.Context, rawURL string) (*extractor.FetchedContent, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	client := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetching %q: HTTP %d", rawURL, resp.StatusCode)
	}

	// Read the full body so we can fall back to tag stripping if needed.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Parse with readability.
	article, err := readability.FromReader(strings.NewReader(string(bodyBytes)), parsedURL)

	var title, rawText string
	if err == nil && len(article.TextContent) >= minReadableChars {
		title = article.Title
		rawText = article.TextContent
	} else {
		// Readability failed or returned too little content — strip HTML tags.
		rawText = stripHTMLTags(string(bodyBytes))
		if err == nil {
			title = article.Title
		}
	}

	return &extractor.FetchedContent{
		URL:         rawURL,
		ContentType: "article",
		Title:       title,
		RawText:     rawText,
		FetchedAt:   time.Now(),
	}, nil
}

// stripHTMLTags removes HTML tags from a string using a simple state machine.
// It removes everything between '<' and '>' characters.
func stripHTMLTags(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))

	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf.WriteRune(r)
		}
	}

	return buf.String()
}
