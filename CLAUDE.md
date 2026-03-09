# CLAUDE.md — recap

This file is the technical implementation guide for Claude (and any developer) working on the recap codebase. Read this before writing any code.

---

## Project Overview

**recap** is a CLI + web app that extracts structured, user-defined fields from any URL using an LLM. The CLI and backend are written in **Go**. The frontend is a **Next.js** app. They share a common HTTP API served by the Go backend.

For full product context, user flows, and UX decisions, read `SPEC.md` first.

---

## Repository Structure

```
recap/
├── CLAUDE.md                  # This file
├── SPEC.md                    # Product specification
├── README.md
├── go.mod
├── go.sum
│
├── cmd/
│   └── recap/
│       └── main.go            # CLI entrypoint (Cobra root command)
│
├── internal/
│   ├── extractor/             # Core extraction engine
│   │   ├── extractor.go       # Main Extract() function
│   │   ├── prompt.go          # LLM prompt construction
│   │   └── schema.go          # Field type definitions + validation
│   │
│   ├── fetcher/               # URL fetching and content extraction
│   │   ├── fetcher.go         # Route to correct fetcher by URL type
│   │   ├── http.go            # Generic HTTP fetch + Readability
│   │   ├── youtube.go         # YouTube transcript/metadata
│   │   ├── twitter.go         # X/Twitter post fetching
│   │   ├── reddit.go          # Reddit JSON API
│   │   ├── github.go          # GitHub REST API
│   │   └── pdf.go             # PDF text extraction
│   │
│   ├── template/              # Template loading and management
│   │   ├── template.go        # Template struct + loader
│   │   ├── builtin.go         # Embedded built-in templates
│   │   └── store.go           # Read/write from ~/.recap/templates/
│   │
│   ├── server/                # HTTP API server
│   │   ├── server.go          # Gin router setup
│   │   ├── handlers.go        # Route handlers
│   │   └── middleware.go      # CORS, logging, rate limiting
│   │
│   └── config/
│       └── config.go          # Config loading from env + ~/.recap/config.yaml
│
├── web/                       # Next.js frontend
│   ├── app/
│   │   ├── page.tsx           # Main extractor UI
│   │   ├── history/page.tsx
│   │   ├── templates/page.tsx
│   │   └── api/               # Next.js API routes proxy to Go backend
│   ├── components/
│   ├── lib/
│   │   └── api.ts             # Typed API client for the Go backend
│   └── package.json
│
├── templates/                 # Built-in template JSON files (embedded)
│   ├── news-article.json
│   ├── social-post.json
│   ├── youtube-video.json
│   ├── research-paper.json
│   └── product-review.json
│
└── Makefile
```

---

## Go Module

```
module github.com/yourusername/recap

go 1.22
```

Key dependencies:
```
github.com/spf13/cobra                  # CLI framework
github.com/gin-gonic/gin                # HTTP server
github.com/anthropics/anthropic-sdk-go  # Anthropic API client
github.com/go-shiori/go-readability     # Article text extraction
gopkg.in/yaml.v3                        # Config file parsing
github.com/tidwall/gjson               # JSON parsing for API responses
```

---

## Core Data Types

Define these in `internal/extractor/schema.go`. All other packages import from here.

```go
// FieldType represents the expected output type of an extracted field
type FieldType string

const (
    FieldTypeText      FieldType = "text"
    FieldTypeList      FieldType = "list"
    FieldTypeEnum      FieldType = "enum"
    FieldTypeNumber    FieldType = "number"
    FieldTypeBoolean   FieldType = "boolean"
    FieldTypeDate      FieldType = "date"
    FieldTypeParagraph FieldType = "paragraph"
)

// Field defines a single extraction target
type Field struct {
    Label   string    `json:"label"`
    Type    FieldType `json:"type"`
    Options []string  `json:"options,omitempty"` // Only for enum type
}

// Template is a named, reusable set of fields
type Template struct {
    Name        string  `json:"name"`
    Description string  `json:"description,omitempty"`
    Fields      []Field `json:"fields"`
}

// FetchedContent is the result of fetching a URL
type FetchedContent struct {
    URL         string    `json:"url"`
    ContentType string    `json:"content_type"` // "article", "tweet", "youtube", etc.
    Title       string    `json:"title,omitempty"`
    RawText     string    `json:"raw_text"`
    FetchedAt   time.Time `json:"fetched_at"`
}

// ExtractionRequest is the input to Extract()
type ExtractionRequest struct {
    URL    string  `json:"url"`
    Fields []Field `json:"fields"`
    Format string  `json:"format"` // "json", "markdown", "csv"
}

// ExtractionResult is the output of Extract()
type ExtractionResult struct {
    URL         string         `json:"url"`
    ContentType string         `json:"content_type"`
    ExtractedAt time.Time      `json:"extracted_at"`
    Data        map[string]any `json:"data"`
    RawContent  string         `json:"raw_content"`
    Model       string         `json:"model"`
    TokensUsed  int            `json:"tokens_used"`
}
```

---

## Extraction Engine (`internal/extractor/extractor.go`)

The `Extract()` function is the core of the application. It:

1. Calls `fetcher.Fetch(url)` to get `FetchedContent`
2. Calls `prompt.Build(content, fields)` to construct the LLM prompt
3. Calls the Anthropic API with the prompt
4. Parses the JSON response into `map[string]any`
5. Validates each field against its declared type
6. Returns an `ExtractionResult`

```go
func Extract(ctx context.Context, req ExtractionRequest, client *anthropic.Client) (*ExtractionResult, error)
```

**Critical implementation details:**

- Always request JSON output from the model using a structured system prompt.
- The system prompt must instruct the model to return a flat JSON object where keys are the field labels (snake_cased) and values match the declared field types.
- For `enum` fields, pass the allowed values explicitly in the prompt. The model must return one of them exactly.
- For `list` fields, the model must return a JSON array of strings.
- For `boolean` fields, the model must return `true` or `false` (not `"yes"`/`"no"`).
- If a field cannot be extracted from the content, the model should return `null` for that key. Never omit the key.
- Parse the model's JSON response and validate each field against its declared type. If a field is the wrong type, attempt a coercion. If coercion fails, set it to `null` and log a warning — do not fail the whole job.

---

## Prompt Construction (`internal/extractor/prompt.go`)

The system prompt must be precise. Here is the canonical structure:

```
You are a structured data extraction assistant. You will be given the text content
of a web page and a list of fields to extract. Your job is to extract the requested
information and return it as a single, flat JSON object.

Rules:
- Return ONLY a valid JSON object. No preamble, no markdown fences, no explanation.
- Every field in the schema must appear as a key in the output, even if the value is null.
- Keys must be the field labels converted to snake_case.
- Respect the type constraints for each field:
  - text / paragraph: a string
  - list: an array of strings
  - enum: exactly one of the provided options as a string
  - number: a numeric value (no units, no quotes)
  - boolean: true or false
  - date: an ISO 8601 date string, or null if not found
- If a field cannot be determined from the content, return null for that key.
- Do not infer or fabricate information that is not present in the content.

Schema:
{{fields_json}}

Content:
{{content}}
```

The `{{fields_json}}` placeholder is replaced with a JSON array of the field definitions. The `{{content}}` placeholder is replaced with the fetched text. If content exceeds the model's context limit, truncate intelligently: keep the beginning and end of the content, summarize the middle. Target a maximum of 80,000 characters of raw content.

---

## Fetcher (`internal/fetcher/fetcher.go`)

The `Fetch()` function routes to the correct fetcher based on the URL:

```go
func Fetch(ctx context.Context, rawURL string) (*FetchedContent, error)
```

**Routing logic (in order):**
1. Parse the URL
2. Check hostname against known patterns:
   - `x.com`, `twitter.com` → `twitter.Fetch()`
   - `youtube.com`, `youtu.be` → `youtube.Fetch()`
   - `reddit.com` → `reddit.Fetch()`
   - `github.com` → `github.Fetch()`
   - `.pdf` extension or PDF Content-Type → `pdf.Fetch()`
3. Default → `http.Fetch()` with Readability extraction

**Generic HTTP fetcher (`internal/fetcher/http.go`):**
- Set a realistic `User-Agent` header
- Follow redirects (max 5)
- Enforce a 30-second timeout
- Use `go-readability` to extract article text from HTML
- Return both the extracted article text and the page title
- If Readability fails to extract meaningful content (less than 200 chars), return the raw body text stripped of HTML tags as a fallback

**YouTube fetcher (`internal/fetcher/youtube.go`):**
- Extract the video ID from the URL
- Use the YouTube Data API v3 to fetch video metadata (title, channel, publish date, duration, description)
- Fetch the auto-generated captions/transcript via the YouTube Transcript API or by parsing the `timedtext` endpoint
- If no transcript is available, return the description + metadata only

**Twitter/X fetcher (`internal/fetcher/twitter.go`):**
- Use Twitter API v2 Bearer Token if configured (`RECAP_TWITTER_BEARER_TOKEN`)
- Extract tweet ID from URL, fetch tweet text, author info, created_at, metrics
- Fallback: fetch the page with a Googlebot User-Agent and extract tweet text from meta tags

---

## CLI (`cmd/recap/main.go`)

Use **Cobra** for the CLI. The root command is `recap`.

### Commands

```
recap run <url> [flags]
  --fields, -f     Comma-separated field definitions. Format: "label:type" or "label:type[opt1,opt2]"
  --template, -t   Template name or path to a .json template file
  --format         Output format: json (default), markdown, csv
  --show-raw       Also print the raw fetched content
  --output, -o     Write output to a file instead of stdout

recap template list
recap template show <name>
recap template save <name> --fields "..."
recap template delete <name>
recap template import <file.json>
recap template export <name> [--output file.json]

recap serve [flags]
  --port    Port to listen on (default: 8080)
  --host    Host to bind to (default: 127.0.0.1)

recap config set <key> <value>
recap config show
```

### Field Flag Parsing

The `--fields` flag accepts a comma-separated string. Parse it into `[]Field`:
- `"author"` → `{Label: "author", Type: FieldTypeText}`
- `"author:text"` → same, explicit type
- `"takeaways:list"` → `{Label: "takeaways", Type: FieldTypeList}`
- `"sentiment:enum[positive,negative,neutral]"` → `{Label: "sentiment", Type: FieldTypeEnum, Options: ["positive", "negative", "neutral"]}`

If both `--fields` and `--template` are provided, merge them: template fields come first, then additional fields from the flag. Deduplicate by label (case-insensitive); later entries win.

### Output Formatting

**JSON** (default): Pretty-printed `ExtractionResult` JSON to stdout.

**Markdown**: Format as a Markdown document with `## field_label` headers and appropriate formatting per type.

**CSV**: One row. Headers are field labels. Lists are joined with ` | `.

---

## HTTP API Server (`internal/server/`)

Use **Gin** for the HTTP server.

### Routes

```
GET  /api/v1/health
POST /api/v1/extract
GET  /api/v1/templates
GET  /api/v1/templates/:name
POST /api/v1/templates
DELETE /api/v1/templates/:name
```

### `POST /api/v1/extract`

Request body:
```json
{
  "url": "string",
  "fields": [{ "label": "string", "type": "string", "options": ["string"] }],
  "template": "string (optional)",
  "format": "json | markdown | csv"
}
```

- If `template` is provided, load template fields and merge with any provided `fields`.
- Call `extractor.Extract()`.
- Return `ExtractionResult` as JSON. Include a `formatted` string field with the result rendered in the requested format.

### CORS

Allow requests from `localhost:3000` in development. Read `RECAP_ALLOWED_ORIGINS` (comma-separated) for production overrides.

### Error Responses

```json
{
  "error": "human-readable message",
  "code": "FETCH_FAILED | EXTRACT_FAILED | INVALID_REQUEST | TEMPLATE_NOT_FOUND"
}
```

---

## Configuration

Config is loaded in this priority order (highest wins):
1. Environment variables (`RECAP_ANTHROPIC_KEY`, `RECAP_PORT`, etc.)
2. `~/.recap/config.yaml`
3. `.recap/config.yaml` in current working directory
4. Defaults

`~/.recap/config.yaml` structure:
```yaml
anthropic_api_key: sk-ant-...
model: claude-sonnet-4
max_tokens: 4096
default_format: json
server:
  port: 8080
  host: 127.0.0.1
```

Use `os.UserHomeDir()` for `~` expansion. Never log values that may contain API keys.

---

## Frontend (`web/`)

Next.js 14 App Router. Calls the Go backend API. `NEXT_PUBLIC_API_URL` env var sets the backend base URL (defaults to `http://localhost:8080`).

### Typed API Client (`web/lib/api.ts`)

```typescript
export interface Field {
  label: string
  type: "text" | "list" | "enum" | "number" | "boolean" | "date" | "paragraph"
  options?: string[]
}

export interface ExtractionRequest {
  url: string
  fields?: Field[]
  template?: string
  format?: "json" | "markdown" | "csv"
}

export interface ExtractionResult {
  url: string
  content_type: string
  extracted_at: string
  data: Record<string, unknown>
  raw_content: string
  model: string
  tokens_used: number
}

export async function extract(req: ExtractionRequest): Promise<ExtractionResult>
export async function getTemplates(): Promise<Template[]>
export async function getTemplate(name: string): Promise<Template>
```

### State Management

Use **Zustand** for global state:
- `extractionStore` — current job (url, fields, loading status, result)
- `historyStore` — past extractions, persisted to `localStorage`

### Key Components

```
components/
├── FieldBuilder.tsx       # Add/remove/edit fields inline
├── TemplateSelector.tsx   # Dropdown + template preview
├── ResultView.tsx         # Fields / JSON / Markdown / Raw tabs
├── ExtractionForm.tsx     # URL input + field builder + run button
└── HistoryList.tsx        # Past extractions with re-run support
```

---

## Development Workflow

### Setup
```bash
go mod download
cd web && npm install
export RECAP_ANTHROPIC_KEY=sk-ant-...
```

### Running locally
```bash
# Terminal 1: Go backend
go run ./cmd/recap serve

# Terminal 2: Next.js frontend
cd web && npm run dev

# CLI directly
go run ./cmd/recap run "https://example.com" --fields "title,author,summary:paragraph"
```

### Makefile targets
```
build        Build binary to ./bin/recap
dev-server   Run Go server with hot reload (air)
dev-web      Run Next.js dev server
test         Run Go tests
lint         golangci-lint
fmt          gofmt + goimports
build-all    Cross-compile linux/darwin/windows, amd64+arm64
```

---

## Testing

- **Unit tests**: `prompt.go` (prompt output per field combination), `schema.go` (type coercion), field flag parser
- **Integration tests**: `extractor.go` with mocked Anthropic client + fetcher — verify structure of output given known inputs
- **HTTP tests**: API route handlers using `httptest`
- **No live network calls in CI**: Use recorded fixtures for fetcher tests

Test files live alongside source (`extractor_test.go` next to `extractor.go`).

---

## Constraints and Gotchas

1. **Never log API keys.** Redact `anthropic_api_key` in all logging and debug output.

2. **Content length truncation.** Truncate raw content to ~80,000 characters before sending to the model. Keep the beginning and end; drop the middle. Most articles front-load the important content.

3. **Do not stream extraction responses.** Use synchronous API calls. Streaming complicates JSON parsing and offers no UX benefit for this use case.

4. **Field label normalization.** Normalize labels to `snake_case` when building prompts and parsing responses. Display the original user-provided label everywhere else.

5. **Template + field merging.** Deduplicate by label, case-insensitively. If the same label appears in both template and `--fields`, the `--fields` version wins.

6. **Do not embed the frontend in the Go binary in v1.** Keep them as separate processes. This simplifies development and deployment iteration.

7. **Cross-platform paths.** Use `os.UserHomeDir()` for home directory resolution. Never hardcode `/tmp` or Unix-style paths. The binary must work on macOS, Linux, and Windows.
