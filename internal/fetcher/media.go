package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/humzakhan/recap/internal/extractor"
)

const (
	maxDownloadSize    = 100 * 1024 * 1024 // 100MB
	maxWhisperFileSize = 25 * 1024 * 1024  // 25MB
	downloadTimeout    = 5 * time.Minute
	connectTimeout     = 30 * time.Second
	whisperTimeout     = 5 * time.Minute
	whisperAPIURL      = "https://api.openai.com/v1/audio/transcriptions"
)

// fetchMedia downloads a video or audio file from a URL and transcribes it
// using the OpenAI Whisper API.
func fetchMedia(ctx context.Context, rawURL string, mediaType string) (*extractor.FetchedContent, error) {
	// Extract filename from URL for the title.
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	title := path.Base(parsed.Path)
	if title == "." || title == "/" {
		title = "unknown"
	}

	// Download the file to a temp location.
	downloadCtx, downloadCancel := context.WithDeadline(ctx, time.Now().Add(downloadTimeout))
	defer downloadCancel()

	tempFile, err := downloadFile(downloadCtx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	defer os.Remove(tempFile)

	audioPath := tempFile

	// If video, extract audio using ffmpeg.
	if mediaType == "video" {
		extracted, err := extractAudio(ctx, tempFile)
		if err != nil {
			return nil, fmt.Errorf("failed to extract audio from video: %w", err)
		}
		defer os.Remove(extracted)
		audioPath = extracted
	}

	// Transcribe the audio.
	transcript, err := transcribeAudio(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	return &extractor.FetchedContent{
		URL:         rawURL,
		ContentType: mediaType,
		Title:       title,
		RawText:     transcript,
		FetchedAt:   time.Now(),
	}, nil
}

// downloadFile downloads a file from a URL to a temporary file.
// It enforces a 30-second connection timeout and a 100MB size limit.
func downloadFile(ctx context.Context, rawURL string) (string, error) {
	// Create an HTTP client with a connection timeout.
	transport := &http.Transport{
		ResponseHeaderTimeout: connectTimeout,
	}
	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check Content-Length before downloading.
	if resp.ContentLength > maxDownloadSize {
		return "", fmt.Errorf("file too large: %d bytes exceeds 100MB limit", resp.ContentLength)
	}

	// Determine extension from URL.
	ext := path.Ext(path.Base(rawURL))
	if ext == "" {
		ext = ".tmp"
	}

	tmpFile, err := os.CreateTemp("", "recap-media-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Use a LimitReader to enforce the max size during download.
	limited := io.LimitReader(resp.Body, maxDownloadSize+1)
	n, err := io.Copy(tmpFile, limited)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	if n > maxDownloadSize {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("file too large: exceeds 100MB limit")
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// transcribeAudio sends an audio file to the OpenAI Whisper API for transcription.
func transcribeAudio(ctx context.Context, audioPath string) (string, error) {
	apiKey := os.Getenv("RECAP_OPENAI_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("audio/video transcription requires RECAP_OPENAI_KEY to be set")
	}

	// Check file size against Whisper's 25MB limit.
	info, err := os.Stat(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat audio file: %w", err)
	}
	if info.Size() > maxWhisperFileSize {
		return "", fmt.Errorf("audio file exceeds 25MB Whisper API limit")
	}

	// Open the audio file.
	audioFile, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	// Build the multipart form request.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the audio file.
	part, err := writer.CreateFormFile("file", path.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, audioFile); err != nil {
		return "", fmt.Errorf("failed to write audio to form: %w", err)
	}

	// Add model field.
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	// Add response_format field.
	if err := writer.WriteField("response_format", "text"); err != nil {
		return "", fmt.Errorf("failed to write response_format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the HTTP request with a 5-minute timeout.
	whisperCtx, cancel := context.WithTimeout(ctx, whisperTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(whisperCtx, http.MethodPost, whisperAPIURL, &body)
	if err != nil {
		return "", fmt.Errorf("failed to create transcription request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Whisper API error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return strings.TrimSpace(string(respBody)), nil
}

// extractAudio uses ffmpeg to extract audio from a video file.
// Returns the path to the extracted .mp3 file. The caller is responsible for cleanup.
func extractAudio(ctx context.Context, videoPath string) (string, error) {
	// Check that ffmpeg is available.
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", fmt.Errorf("ffmpeg is required for video processing. Install: https://ffmpeg.org/download.html")
	}

	// Create temp output path with .mp3 extension.
	outputFile, err := os.CreateTemp("", "recap-audio-*.mp3")
	if err != nil {
		return "", fmt.Errorf("failed to create temp audio file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()

	// Run ffmpeg to extract audio.
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", videoPath, "-vn", "-acodec", "libmp3lame", "-q:a", "4", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}

	return outputPath, nil
}
