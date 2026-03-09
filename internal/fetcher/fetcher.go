package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/humzakhan/recap/internal/extractor"
	"github.com/humzakhan/recap/internal/log"
)

// Fetch routes to the correct fetcher based on the URL.
func Fetch(ctx context.Context, rawURL string) (*extractor.FetchedContent, error) {
	log.Debug("fetcher: starting fetch for %q", rawURL)

	// 1. Check if it's a local path.
	if isLocalPath(rawURL) {
		localPath := rawURL
		if strings.HasPrefix(rawURL, "file://") {
			localPath = strings.TrimPrefix(rawURL, "file://")
		}
		log.Debug("fetcher: detected local path %q", localPath)
		content, err := fetchLocal(ctx, localPath)
		if err != nil {
			return nil, err
		}
		log.Debug("fetcher: local fetch complete, %d chars extracted", len(content.RawText))
		return content, nil
	}

	// 2. Parse the URL.
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// 3. Do a HEAD request with a 10-second timeout to check Content-Type.
	log.Debug("fetcher: sending HEAD request to detect content type")
	contentType, headErr := headContentType(ctx, rawURL)
	if headErr != nil {
		log.Debug("fetcher: HEAD request failed (%v), falling back to extension routing", headErr)
	} else {
		log.Debug("fetcher: HEAD Content-Type: %q", contentType)
	}

	// 4. Route by Content-Type or file extension.
	if headErr == nil {
		if result := routeByContentType(ctx, rawURL, parsed, contentType); result != nil {
			log.Debug("fetcher: routing by Content-Type")
			return result()
		}
	}

	// 5. If HEAD failed or Content-Type didn't match, fall back to extension-based routing.
	if result := routeByExtension(ctx, rawURL, parsed); result != nil {
		log.Debug("fetcher: routing by file extension")
		return result()
	}

	// 6. Default to HTTP fetch.
	log.Debug("fetcher: using default HTTP fetch with readability extraction")
	content, err := fetchHTTP(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	log.Debug("fetcher: HTTP fetch complete, title=%q, %d chars extracted", content.Title, len(content.RawText))
	return content, nil
}

// isLocalPath checks whether the given string looks like a local file path
// rather than a remote URL.
func isLocalPath(s string) bool {
	if strings.HasPrefix(s, "file://") {
		return true
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") || strings.HasPrefix(s, "~") {
		return true
	}
	if !strings.Contains(s, "://") {
		// Could be a bare filename — but only if it doesn't look like a hostname.
		// We treat it as local only if it has no scheme at all.
		// However, something like "example.com" should NOT be local.
		// We check: no scheme means local only if it starts with /, ., or ~
		// which is already handled above. So this branch is for strings
		// that truly have no scheme and don't start with those prefixes.
		// Per spec: "no '://' → local" so we follow that.
		return true
	}
	return false
}

// headContentType performs a HEAD request and returns the Content-Type header value.
func headContentType(ctx context.Context, rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	return resp.Header.Get("Content-Type"), nil
}

// fetcherFunc is a deferred fetcher call.
type fetcherFunc func() (*extractor.FetchedContent, error)

// routeByContentType returns a fetcherFunc if the Content-Type matches a known type,
// or nil if no match.
func routeByContentType(ctx context.Context, rawURL string, parsed *url.URL, contentType string) fetcherFunc {
	ct := strings.ToLower(contentType)

	if strings.Contains(ct, "application/pdf") {
		return func() (*extractor.FetchedContent, error) { return fetchPDF(ctx, rawURL) }
	}
	if strings.HasPrefix(ct, "video/") {
		return func() (*extractor.FetchedContent, error) { return fetchMedia(ctx, rawURL, "video") }
	}
	if strings.HasPrefix(ct, "audio/") {
		return func() (*extractor.FetchedContent, error) { return fetchMedia(ctx, rawURL, "audio") }
	}

	return nil
}

// routeByExtension returns a fetcherFunc based on the URL's file extension,
// or nil if no match.
func routeByExtension(ctx context.Context, rawURL string, parsed *url.URL) fetcherFunc {
	ext := strings.ToLower(path.Ext(parsed.Path))

	switch ext {
	case ".pdf":
		return func() (*extractor.FetchedContent, error) { return fetchPDF(ctx, rawURL) }
	case ".mp4", ".mov", ".webm", ".avi":
		return func() (*extractor.FetchedContent, error) { return fetchMedia(ctx, rawURL, "video") }
	case ".mp3", ".wav", ".m4a", ".ogg", ".flac":
		return func() (*extractor.FetchedContent, error) { return fetchMedia(ctx, rawURL, "audio") }
	}

	return nil
}

// fetchLocal is implemented in local.go.
// fetchPDF is implemented in pdf.go.
// fetchMedia is implemented in media.go.
