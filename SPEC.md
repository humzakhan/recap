# recap — Product Specification

> Extract structured intelligence from any URL on the web.

---

## 1. Core Thesis

The web is full of content, but extracting specific, structured information from it is tedious. You either read the whole thing yourself, paste it into a chat interface and ask questions manually, or write brittle one-off scrapers that break when the page changes.

**recap** is a tool that lets you point at any piece of content on the web — an X post, a news article, a YouTube video, a blog post, a podcast transcript, a research paper — and define *exactly* what you want to know about it. The tool fetches the content, passes it to an LLM, and returns a clean, structured result in the format you asked for.

The key insight is that **the extraction schema lives with the user, not the tool**. recap doesn't assume you want a summary. It asks you what you want. You define the fields, the types, and optionally the constraints. recap does the rest.

This works as a CLI for developers and power users, and as a web UI for anyone who wants a more visual workflow.

---

## 2. Target Users

| User | Context | Primary Interface |
|---|---|---|
| Developer / data engineer | Automating content pipelines, research workflows | CLI + API |
| Researcher / analyst | Extracting structured data from many sources | Web UI + CSV export |
| Content team | Monitoring competitors, summarizing industry news | Web UI |
| Power user / indie hacker | Personal knowledge management, link analysis | CLI or Web UI |

---

## 3. Core Concepts

### 3.1 Extraction Job

An extraction job is the atomic unit of recap. It consists of:

- **URL** — The target content. Any publicly accessible URL.
- **Fields** — A list of named fields with types the user wants extracted.
- **Output format** — JSON (default), Markdown, or CSV.
- **Template** (optional) — A named, reusable set of fields.

### 3.2 Field Types

Fields define what the LLM should extract and how it should be typed in the output.

| Type | Description | Example |
|---|---|---|
| `text` | Short freeform string | `"author"`, `"headline"` |
| `list` | Array of strings | `"key_takeaways"`, `"tags"` |
| `enum` | One of a predefined set of values | `"sentiment: positive/negative/neutral"` |
| `number` | Numeric value | `"word_count"`, `"year_published"` |
| `boolean` | True/false | `"is_opinion"`, `"has_paywall"` |
| `date` | ISO 8601 date string | `"published_at"` |
| `paragraph` | Longer freeform text | `"summary"`, `"main_argument"` |

### 3.3 Templates

Templates are named, reusable field configurations. They can be:

- **Built-in** — Shipped with recap (News Article, Social Post, YouTube Video, Research Paper, Product Review)
- **User-defined** — Saved locally (`~/.recap/templates/`) or in the project directory (`.recap/templates/`)
- **Exported/shared** — As JSON files that can be committed to a repo or shared with a team

### 3.4 Content Types

recap detects and handles the following content types automatically:

| Type | Detection | Fetching Strategy |
|---|---|---|
| Standard web page / article | Default | HTTP fetch + Readability extraction |
| X / Twitter post | `x.com`, `twitter.com` | API or scrape fallback |
| YouTube video | `youtube.com`, `youtu.be` | YouTube Data API → transcript |
| Reddit post/thread | `reddit.com` | Reddit JSON API |
| GitHub repo / issue / PR | `github.com` | GitHub REST API |
| PDF | `.pdf` extension or `Content-Type` | PDF text extraction |
| Substack / Ghost / Medium | Known domains | HTTP fetch + Readability |
| Podcast episode | RSS feed URL or known platforms | Transcript API or audio transcription |

---

## 4. User Flows

### 4.1 CLI Flow

**Basic extraction:**
```
recap run <url> --fields "author,date,summary:paragraph,sentiment:enum[positive,negative,neutral]"
```

**Using a template:**
```
recap run <url> --template news-article
recap run <url> --template ./my-template.json
```

**Saving a template:**
```
recap template save research-paper \
  --fields "title,authors:list,abstract:paragraph,methodology:paragraph,findings:list,year:number"
```

**Batch processing:**
```
recap batch urls.txt --template news-article --output results.csv
```

**Output format:**
```
recap run <url> --template news-article --format json     # default
recap run <url> --template news-article --format markdown
recap run <url> --template news-article --format csv
```

**Running the web server:**
```
recap serve           # starts on :8080
recap serve --port 3001
```

### 4.2 Web UI Flow

1. **Landing / Home** — User arrives at a clean input screen. A URL field is prominent. Below it, fields can be added or a template selected.

2. **Define Extraction** — User pastes a URL and either:
   - Picks a built-in or saved template
   - Adds fields manually (label + type) using an inline field builder
   - Mixes both (start from a template, then add/remove fields)

3. **Run Extraction** — User hits Extract. A loading state shows while the URL is fetched and processed. The URL is shown in the loading state so the user can confirm what's being processed.

4. **View Results** — Results are displayed in a structured card layout, one card per field. Tabs allow switching between:
   - **Fields** — Visual structured view
   - **JSON** — Raw JSON output, copyable
   - **Markdown** — Formatted markdown output
   - **Raw** — The fetched text that was passed to the model

5. **Export / Copy** — User can:
   - Copy the JSON to clipboard
   - Download as `.json`, `.md`, or `.csv`
   - Save the extraction to history (automatic)
   - Save the field configuration as a template

6. **History** — A sidebar or dedicated page showing past extractions, searchable by URL, date, or template. Re-running an extraction on the same URL produces a diff view if content has changed.

7. **Templates** — A management page for user templates. Templates can be created, edited, exported, imported, and shared via a public link.

### 4.3 API Flow (for integrations)

The backend exposes a REST API that both the web UI and external integrations consume.

```
POST /api/v1/extract
{
  "url": "https://...",
  "fields": [
    { "label": "author", "type": "text" },
    { "label": "key_takeaways", "type": "list" },
    { "label": "sentiment", "type": "enum", "options": ["positive", "negative", "neutral"] }
  ],
  "format": "json"
}
```

Response:
```json
{
  "url": "https://...",
  "content_type": "article",
  "extracted_at": "2026-03-08T14:32:00Z",
  "data": {
    "author": "Jane Doe",
    "key_takeaways": ["Point A", "Point B", "Point C"],
    "sentiment": "positive"
  },
  "raw_content": "...",
  "model": "claude-sonnet-4",
  "tokens_used": 1840
}
```

---

## 5. UX Principles

### 5.1 Define-first, extract-second

The user always defines what they want before anything is fetched. This keeps the interface intentional and avoids "generic summary" outputs nobody asked for.

### 5.2 Fields are the API

The field list is the primary interaction primitive in both the CLI and UI. Everything else (templates, output formats, batch mode) is scaffolding around it. Keep the field definition experience frictionless.

### 5.3 Transparency on the raw content

Always surface the raw fetched content (the `raw` tab in the UI, `--show-raw` in CLI). Users should be able to see exactly what the model was given. This builds trust and helps debug bad extractions.

### 5.4 Partial results over failure

If the model can't confidently extract a field, return `null` for that field with an optional `_confidence` annotation. Do not fail the whole job because one field couldn't be extracted.

### 5.5 Templates reduce activation energy

First-time users should see templates prominently. Built-in templates covering the most common use cases (news, social, academic) make the tool immediately useful before the user has designed their own workflow.

### 5.6 CLI and UI share identical semantics

A CLI invocation and a UI extraction should produce byte-for-byte identical output for the same inputs. The UI is a visual wrapper over the same backend, not a different product. This makes it easy to graduate from UI to CLI automation.

---

## 6. Built-in Templates

### `news-article`
```json
[
  { "label": "headline", "type": "text" },
  { "label": "author", "type": "text" },
  { "label": "publication", "type": "text" },
  { "label": "published_at", "type": "date" },
  { "label": "summary", "type": "paragraph" },
  { "label": "key_points", "type": "list" },
  { "label": "sentiment", "type": "enum", "options": ["positive", "negative", "neutral"] },
  { "label": "topics", "type": "list" }
]
```

### `social-post`
```json
[
  { "label": "author_handle", "type": "text" },
  { "label": "author_name", "type": "text" },
  { "label": "posted_at", "type": "date" },
  { "label": "main_claim", "type": "paragraph" },
  { "label": "tone", "type": "enum", "options": ["informational", "opinion", "promotional", "humorous", "controversial"] },
  { "label": "mentions", "type": "list" },
  { "label": "hashtags", "type": "list" }
]
```

### `youtube-video`
```json
[
  { "label": "title", "type": "text" },
  { "label": "channel", "type": "text" },
  { "label": "published_at", "type": "date" },
  { "label": "duration_minutes", "type": "number" },
  { "label": "summary", "type": "paragraph" },
  { "label": "key_takeaways", "type": "list" },
  { "label": "topics_covered", "type": "list" },
  { "label": "is_tutorial", "type": "boolean" }
]
```

### `research-paper`
```json
[
  { "label": "title", "type": "text" },
  { "label": "authors", "type": "list" },
  { "label": "published_at", "type": "date" },
  { "label": "abstract", "type": "paragraph" },
  { "label": "methodology", "type": "paragraph" },
  { "label": "key_findings", "type": "list" },
  { "label": "limitations", "type": "list" },
  { "label": "field", "type": "text" }
]
```

### `product-review`
```json
[
  { "label": "product_name", "type": "text" },
  { "label": "reviewer", "type": "text" },
  { "label": "rating", "type": "number" },
  { "label": "pros", "type": "list" },
  { "label": "cons", "type": "list" },
  { "label": "verdict", "type": "paragraph" },
  { "label": "recommended", "type": "boolean" }
]
```

---

## 7. Error States

| Scenario | Behavior |
|---|---|
| URL unreachable / 404 | Return clear error, suggest checking URL |
| Paywalled content | Return partial result from visible content + note that content may be paywalled |
| Rate limited by source | Retry with backoff, surface error if exhausted |
| Content too long for context window | Chunk content intelligently, extract per-chunk and merge |
| Field not found in content | Return `null` for that field, do not fail the job |
| Invalid URL | Validate before fetching, return immediate error |
| Unsupported content type (e.g. binary) | Return clear error explaining supported types |

---

## 8. Out of Scope (v1)

- Authentication / user accounts (v1 is local-first or self-hosted)
- Real-time / live content monitoring (planned for v2)
- Browser extension
- Mobile app
- Multi-URL comparison in a single job (planned for v2)
- Audio transcription for podcasts (deferred — complexity)
