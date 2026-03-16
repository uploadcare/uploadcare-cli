# Uploadcare CLI

A non-interactive command-line interface for the [Uploadcare](https://uploadcare.com) platform, written in Go. Manage files, projects, webhooks, conversions, and more from your terminal or CI/CD pipelines.

## Features

- **File management** — upload, list, copy, store, and delete files
- **JSON & NDJSON output** — structured output with field filtering and `jq` support
- **Stdin piping** — compose commands for batch operations
- **Dry-run mode** — preview destructive operations before executing
- **Multi-project support** — switch between projects via config or flags
- **AI-friendly** — input sanitization, field masking, NDJSON streaming

## Installation

### From source

Requires Go 1.22+.

```bash
git clone https://github.com/uploadcare/uploadcare-cli.git
cd uploadcare-cli
make build
./bin/uploadcare version
```

## Quick start

```bash
# Set credentials
export UPLOADCARE_PUBLIC_KEY="your-public-key"
export UPLOADCARE_SECRET_KEY="your-secret-key"

# List files
uploadcare file list

# Upload a file
uploadcare file upload photo.jpg

# Get file info as JSON
uploadcare file info <uuid> --json

# Delete all unstored files (piping)
uploadcare file list --page-all --stored false --json uuid \
  | uploadcare file delete --from-stdin
```

## Configuration

The CLI resolves configuration from multiple sources. Higher-priority sources override lower ones.

### Priority order

| Priority | Source | Example |
|----------|--------|---------|
| 1 (highest) | CLI flags | `--public-key pk --secret-key sk` |
| 2 | Environment variables | `UPLOADCARE_PUBLIC_KEY=pk` |
| 3 | Named project (`--project` flag or `UPLOADCARE_PROJECT` env) | `--project "Staging"` |
| 4 | Default project from config file | `default_project: "My App"` |
| 5 (lowest) | Top-level keys in config file | `public_key: pk` |

If either `--public-key` or `--secret-key` is set via flags or environment variables, named project lookup is skipped entirely. This prevents misconfigured environments from silently targeting the wrong project.

### Config file

Location: `~/.uploadcare/config.yaml`

**Minimal config:**

```yaml
public_key: "demopublickey"
secret_key: "demosecretkey"
```

**Full config with multiple projects:**

```yaml
# Account-level token (for project management commands)
project_api_token: "your-bearer-token"

# Default project for commands that need public_key/secret_key
default_project: "My App"

# Named projects
projects:
  "My App":
    public_key: "abc123"
    secret_key: "secret..."
    cdn_base: "https://my-custom-cdn.example.com"  # optional per-project override
  "Staging":
    public_key: "def456"
    secret_key: "secret..."

# Optional global overrides
rest_api_base: "https://api.uploadcare.com"
upload_api_base: "https://upload.uploadcare.com"
# Global cdn_base fallback — used when neither the flag, env var, nor the
# resolved project entry provides a cdn_base. When omitted entirely, the
# CDN base is auto-computed from the project's public key.
# cdn_base: "https://global-cdn-override.example.com"
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `UPLOADCARE_PUBLIC_KEY` | API public key |
| `UPLOADCARE_SECRET_KEY` | API secret key |
| `UPLOADCARE_PROJECT_API_TOKEN` | Account-level bearer token |
| `UPLOADCARE_PROJECT` | Named project to use from config |
| `UPLOADCARE_VERBOSE` | Enable verbose HTTP logging (`1` or `true`) |
| `UPLOADCARE_REST_API_BASE` | Override REST API base URL |
| `UPLOADCARE_UPLOAD_API_BASE` | Override Upload API base URL |
| `UPLOADCARE_CDN_BASE` | Override CDN base URL |
| `UPLOADCARE_PROJECT_API_BASE` | Override Project API base URL |
| `NO_COLOR` | Disable colored output |

### Project selection

```bash
# Uses default_project from config
uploadcare file list

# Select a specific project
uploadcare --project "Staging" file list

# Same via environment variable
UPLOADCARE_PROJECT="Staging" uploadcare file list

# Override with explicit keys (skips project lookup)
uploadcare --public-key pk --secret-key sk file list
```

### Authentication

The CLI works with three Uploadcare APIs, each using different credentials:

| API | Auth method | Credentials |
|-----|-------------|-------------|
| REST API | HMAC signature | `public_key` + `secret_key` |
| Upload API | Simple auth | `public_key` only |
| Project API | Bearer token | `project_api_token` |

Commands validate that the required credentials are present before executing. Missing credentials produce a clear error with instructions on how to set them.

## Commands

```
uploadcare
├── file
│   ├── list              List files in project
│   ├── info              Get file details
│   ├── upload            Upload local file(s)
│   ├── upload-from-url   Upload file from URL
│   ├── store             Store file(s)
│   ├── delete            Delete file(s)
│   ├── local-copy        Copy file within Uploadcare storage
│   └── remote-copy       Copy file to remote storage
├── version               Print CLI version
└── completion            Generate shell completions
```

Additional command groups are planned: `metadata`, `group`, `convert`, `addon`, `webhook`, `project`, and `config`.

### Global flags

| Flag | Description |
|------|-------------|
| `--public-key` | API public key |
| `--secret-key` | API secret key |
| `--project-api-token` | Account-level bearer token |
| `--project` | Named project from config |
| `--json [fields]` | JSON output; optional comma-separated field list |
| `--jq <expr>` | Apply jq expression (implies `--json`) |
| `-q, --quiet` | Suppress non-error output |
| `-v, --verbose` | Log HTTP requests/responses to stderr |
| `--no-color` | Disable colored output |

## Output modes

**Human-readable** (default) — tabular output to stdout:

```
$ uploadcare file list --limit 3
UUID                                  SIZE      FILENAME       STORED   UPLOADED
a1b2c3d4-e5f6-7890-abcd-ef1234567890 1258000   photo.jpg      true     2026-03-01T00:00:00Z
b2c3d4e5-f6a7-8901-bcde-f12345678901 348160    document.pdf   false    2026-03-02T00:00:00Z
```

**JSON** — activated with `--json`, supports field filtering:

```bash
uploadcare file info <uuid> --json uuid,size,filename
```

**NDJSON** — one JSON object per line with `--page-all`:

```bash
uploadcare file list --page-all --json uuid,size
```

**Verbose** — HTTP details on stderr (combine with any mode):

```
$ uploadcare file list --verbose
--> GET https://api.uploadcare.com/files/?limit=100
<-- 200 OK (127ms)
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API error or unexpected failure |
| 2 | Usage error (invalid flags, bad input) |
| 3 | Authentication/configuration error |

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint
```

## License

[MIT](LICENSE)
