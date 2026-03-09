# recap

Point recap at any URL — a news article, a YouTube video, a PDF, a blog post — and tell it exactly what you want to know. Instead of reading the whole thing, copying text into a chat window, or writing a scraper that breaks next week, you define the fields you care about (author, key takeaways, sentiment, publication date) and recap extracts them into clean, structured data.

This is useful for researchers pulling structured data from dozens of papers, content teams monitoring competitor blogs, developers building data pipelines over web content, or anyone who's tired of manually reading things just to extract three facts.

recap works as a **CLI** for scripting and automation, an **HTTP API** for integrations, and a **web UI** for visual workflows. All three interfaces produce identical output for the same inputs.

## Quick Start

### Prerequisites

- **Go 1.25+** — [install](https://go.dev/dl/)
- **Node.js 20+** — [install](https://nodejs.org/) (only needed for the web UI)
- **An Anthropic API key** — [get one](https://console.anthropic.com/settings/keys)
- **ffmpeg** (optional) — only needed if you want to transcribe video/audio files

### Install & Build

```bash
git clone https://github.com/humzakhan/recap.git
cd recap

# Install Go dependencies
go mod download

# Build the binary
make build
# Binary is at ./bin/recap

# Install frontend dependencies (optional, for web UI)
cd web && npm install && cd ..
```

### Set Your API Key

```bash
export RECAP_ANTHROPIC_KEY=sk-ant-...
```

Or create `~/.recap/config.yaml`:

```yaml
anthropic_api_key: sk-ant-...
```

### Run Your First Extraction

```bash
# Extract specific fields from any URL
./bin/recap run "https://example.com/article" \
  --fields "title,author,summary:paragraph,tags:list"

# Use a built-in template
./bin/recap run "https://example.com/article" --template news-article

# Output as markdown
./bin/recap run "https://example.com" --template research-paper --format markdown
```

### Start the Web UI

```bash
# Start both the Go backend and Next.js frontend
make dev
```

Then open [http://localhost:3000](http://localhost:3000). The backend API runs on port 8080.

## CLI Reference

### `recap run <url>`

Extract structured data from a URL.

```
--fields, -f    Field definitions: "title,author:text,tags:list,sentiment:enum[pos,neg,neutral]"
--template, -t  Template name or path to a .json template file
--format        Output format: json (default), markdown, csv
--show-raw      Print the raw fetched content alongside the extraction
--output, -o    Write output to a file instead of stdout
```

**Field syntax:**
- `author` — defaults to type `text`
- `author:text` — explicit type
- `tags:list` — array of strings
- `rating:number` — numeric value
- `is_opinion:boolean` — true/false
- `published:date` — ISO 8601 date
- `summary:paragraph` — longer freeform text
- `sentiment:enum[positive,negative,neutral]` — constrained to specific values

### `recap template`

```bash
recap template list                                   # List all templates
recap template show news-article                      # Show template fields
recap template save my-template --fields "title,tags:list"  # Save a custom template
recap template delete my-template                     # Delete a custom template
recap template export news-article -o template.json   # Export to file
recap template import template.json                   # Import from file
```

### `recap serve`

Start the HTTP API server.

```bash
recap serve              # Starts on 127.0.0.1:8080
recap serve --port 3001  # Custom port
```

### `recap config`

```bash
recap config show        # Show current config (API key redacted)
recap config set <key> <value>
```

## Built-in Templates

| Template | Fields |
|----------|--------|
| `news-article` | headline, author, publication, published_at, summary, key_points, sentiment, topics |
| `social-post` | author_handle, author_name, posted_at, main_claim, tone, mentions, hashtags |
| `youtube-video` | title, channel, published_at, duration_minutes, summary, key_takeaways, topics_covered, is_tutorial |
| `research-paper` | title, authors, published_at, abstract, methodology, key_findings, limitations, field |
| `product-review` | product_name, reviewer, rating, pros, cons, verdict, recommended |

## API Endpoints

All endpoints are under `/api/v1`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/extract` | Run an extraction |
| `GET` | `/templates` | List all templates |
| `GET` | `/templates/:name` | Get a template by name |
| `POST` | `/templates` | Create a user template |
| `DELETE` | `/templates/:name` | Delete a user template |

**Example extraction request:**

```bash
curl -X POST http://localhost:8080/api/v1/extract \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/article",
    "fields": [
      {"label": "title", "type": "text"},
      {"label": "key_points", "type": "list"},
      {"label": "sentiment", "type": "enum", "options": ["positive", "negative", "neutral"]}
    ]
  }'
```

## Supported Content Types

| Type | How it's detected | What happens |
|------|-------------------|--------------|
| Web pages | Default | Fetched via HTTP, article text extracted with Readability |
| PDF files | `.pdf` extension or `application/pdf` Content-Type | Text extracted from all pages |
| Local files | Path with no scheme (e.g., `./doc.txt`, `~/notes.md`) | Read directly from disk |
| Video files | `.mp4`, `.mov`, `.webm` extension or `video/*` Content-Type | Audio extracted with ffmpeg, transcribed via Whisper |
| Audio files | `.mp3`, `.wav`, `.m4a` extension or `audio/*` Content-Type | Transcribed via Whisper |

For video/audio transcription, set `RECAP_OPENAI_KEY` and ensure `ffmpeg` is installed.

## Project Structure

```
recap/
├── cmd/recap/          # CLI entrypoint (Cobra commands, field parser, formatters)
├── internal/
│   ├── extractor/      # Core extraction engine, LLM prompts, type coercion
│   ├── fetcher/        # URL fetching (HTTP, local files, PDF, media)
│   ├── template/       # Template loading (builtins + user-defined)
│   ├── server/         # Gin HTTP server, handlers, middleware
│   └── config/         # Layered config (env > yaml > defaults)
├── web/                # Next.js 14 frontend
│   ├── src/app/        # Pages (extract, history, templates)
│   ├── src/components/ # React components
│   └── src/lib/        # API client, Zustand stores
├── Makefile
├── SPEC.md             # Product specification
└── CLAUDE.md           # Technical implementation guide
```

## Configuration

Config is loaded in priority order (highest wins):

1. Environment variables (`RECAP_ANTHROPIC_KEY`, `RECAP_MODEL`, `RECAP_PORT`, etc.)
2. `~/.recap/config.yaml`
3. `.recap/config.yaml` in the current directory
4. Defaults

| Variable | Config Key | Default |
|----------|-----------|---------|
| `RECAP_ANTHROPIC_KEY` | `anthropic_api_key` | (required) |
| `RECAP_MODEL` | `model` | `claude-sonnet-4-20250514` |
| `RECAP_MAX_TOKENS` | `max_tokens` | `4096` |
| `RECAP_DEFAULT_FORMAT` | `default_format` | `json` |
| `RECAP_PORT` | `server.port` | `8080` |
| `RECAP_HOST` | `server.host` | `127.0.0.1` |
| `RECAP_OPENAI_KEY` | — | (required for audio/video) |
| `RECAP_ALLOWED_ORIGINS` | — | `http://localhost:3000` |
| `RECAP_RATE_LIMIT` | — | `10` (requests/min for /extract) |

## Development

```bash
# Run everything locally
make dev            # Starts Go server + Next.js dev server

# Run tests
make test           # Go tests with race detection

# Lint
make lint           # go vet

# Format
make fmt            # gofmt + goimports

# Build for all platforms
make build-all      # linux/darwin/windows, amd64/arm64
```

User-defined templates are stored in `~/.recap/templates/` as JSON files.

## License

MIT
