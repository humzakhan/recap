package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	pdflib "github.com/ledongthuc/pdf"

	"github.com/humzakhan/recap/internal/extractor"
)

// fetchPDF fetches and extracts text from a PDF at the given HTTP URL.
func fetchPDF(ctx context.Context, rawURL string) (*extractor.FetchedContent, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pdf: server returned status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "recap-pdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return nil, fmt.Errorf("pdf: failed to write temp file: %w", err)
	}

	text, err := extractPDFText(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}

	return &extractor.FetchedContent{
		URL:         rawURL,
		ContentType: "pdf",
		Title:       "",
		RawText:     text,
		FetchedAt:   time.Now(),
	}, nil
}

// fetchPDFFromReader extracts text from PDF data provided as a byte slice.
// This is used for local PDF files (called from local.go).
func fetchPDFFromReader(data []byte, source string) (*extractor.FetchedContent, error) {
	tmpFile, err := os.CreateTemp("", "recap-pdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil, fmt.Errorf("pdf: failed to write temp file: %w", err)
	}

	text, err := extractPDFText(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}

	return &extractor.FetchedContent{
		URL:         source,
		ContentType: "pdf",
		Title:       "",
		RawText:     text,
		FetchedAt:   time.Now(),
	}, nil
}

// extractPDFText opens a PDF file and extracts plain text from all pages.
func extractPDFText(filePath string) (string, error) {
	f, r, err := pdflib.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	totalPage := r.NumPage()
	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
	}

	result := strings.TrimSpace(buf.String())
	if len(result) == 0 {
		return "", fmt.Errorf("no text content found in PDF (scanned/image PDFs are not supported)")
	}

	return result, nil
}
