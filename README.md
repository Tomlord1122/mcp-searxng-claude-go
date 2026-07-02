# MCP SearXNG Go

A high-performance [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server implementation in Go that provides AI assistants with privacy-focused web search capabilities through [SearXNG](https://docs.searxng.org/).

Built with the [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) maintained in collaboration with Google.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![MCP SDK](https://img.shields.io/badge/MCP--SDK-Official-green)](https://github.com/modelcontextprotocol/go-sdk)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- 🔍 **Web Search**: Powered by SearXNG metasearch engine, curated to ~14 lightweight enabled engines by default (250+ more available, disabled by default)
- 🌐 **URL Content Extraction**: Fetch and convert web pages to Markdown
- 🔒 **Privacy-Focused**: All searches go through your own SearXNG instance
- ⚡ **High Performance**: Built in Go with in-memory caching (configurable TTL and max entries, default 60s / 500 entries)
- 🐳 **Docker Ready**: One-command deployment with docker-compose
- 🛡️ **Secure**: Non-root container execution, size limits, request timeouts

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- Go 1.25+ (for local development)

### Using Docker (Recommended)

1. **Clone the repository:**
   ```bash
   git clone https://github.com/Tomlord1122/mcp-searxng-claude-go.git
   cd mcp-searxng-claude-go
   ```

2. **Create environment file:**
   ```bash
   cp .env.example .env
   echo "SEARXNG_SECRET=$(openssl rand -hex 16)" >> .env
   ```

3. **Start services:**
   ```bash
   # Build and start both SearXNG and MCP server
   docker compose up -d

   # Or use convenience script
   ./start-mcp.sh
   ```

4. **Verify it's running:**
   ```bash
   docker compose logs -f mcp-searxng-go
   ```
5. **Setup your environment to use it in Claude Code**
   Create a file with this name `~/.mcp.json`

   ```json
   {
     "mcpServers": {
       "searxng": {
         "command": "docker",
         "args": ["exec", "-i", "mcp-searxng-go", "/app/mcp-searxng-go"],
         "env": {
           "SEARXNG_URL": "http://searxng:8080"
         }
       }
     }
   }
   ```

   Add this to `~/.claude/settings.json`
   ```json
   {
    "permissions": {
    "allow": [
      "mcp__searxng__web_search",
      "mcp__searxng__url_read"
    ]
    },
    "enabledMcpjsonServers": [
      "searxng"
    ]
    }
 
   ```
### Local Development

1. **Install dependencies:**
   ```bash
   make deps
   ```

2. **Build the binary:**
   ```bash
   make build
   ```

3. **Run locally:**
   ```bash
   # Requires a running SearXNG instance
   SEARXNG_URL=http://localhost:8080 ./mcp-searxng-go
   ```

## MCP Tools

### 1. `web_search`

Performs web searches using SearXNG.

**Parameters:**
- `query` (required): Search query string
- `pageno` (optional): Page number (default: 1)
- `time_range` (optional): Filter by time - "day", "month", or "year"
- `language` (optional): Language code (e.g., "en", "fr", "de", default: "all")
- `safesearch` (optional): Safe search level - "0", "1", or "2" (default: "0")

**Example:**
```json
{
  "query": "Go programming best practices",
  "pageno": 1,
  "language": "en"
}
```

### 2. `url_read`

Reads and converts web page content to Markdown.

**Parameters:**
- `url` (required): URL to fetch
- `startChar` (optional): Starting character position
- `maxLength` (optional): Maximum characters to return
- `section` (optional): Extract specific heading section
- `paragraphRange` (optional): Paragraph range (e.g., "1-5", "10-")
- `readHeadings` (optional): Return only headings (boolean)

**Example:**
```json
{
  "url": "https://example.com/article",
  "maxLength": 5000
}
```

## Integration with Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "searxng": {
      "command": "docker",
      "args": ["exec", "-i", "mcp-searxng-go", "/app/mcp-searxng-go"],
      "env": {
        "SEARXNG_URL": "http://searxng:8080"
      }
    }
  }
}
```

Or if installed locally:

```json
{
  "mcpServers": {
    "searxng": {
      "command": "/usr/local/bin/mcp-searxng-go",
      "env": {
        "SEARXNG_URL": "http://localhost:8080"
      }
    }
  }
}
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SEARXNG_URL` | ✅ Yes | - | SearXNG instance URL (e.g., `http://localhost:8080`) |
| `SEARXNG_SECRET` | Recommended | - | Secret key for SearXNG instance |
| `AUTH_USERNAME` | No | - | Basic auth username for SearXNG |
| `AUTH_PASSWORD` | No | - | Basic auth password for SearXNG |
| `HTTP_PROXY` | No | - | HTTP proxy URL |
| `HTTPS_PROXY` | No | - | HTTPS proxy URL |
| `CACHE_TTL` | No | 60 | URL-read cache time-to-live in seconds |
| `CACHE_MAX_ENTRIES` | No | 500 | Max cached URLs kept in memory (retention cap, prevents unbounded growth) |

### SearXNG Configuration

The SearXNG instance is automatically configured to output JSON format. Custom search engines can be configured in `searxng/config/settings.yml`.

By default this repo ships with a curated set of ~14 enabled engines (google, bing, duckduckgo, brave, startpage, wikipedia, wikidata, github, stackoverflow, askubuntu, superuser, arxiv, pypi, mdn) instead of SearXNG's full default set, to reduce memory/CPU/connection overhead and avoid several upstream-broken engines that otherwise spam startup logs with errors. All other engines remain present but disabled — flip `disabled: true` off for any engine you want to re-enable.

Outgoing request behavior is also tuned down to match this smaller engine set: `request_timeout: 3.0` / `max_request_timeout: 8.0`, `pool_connections: 20`, `pool_maxsize: 10` (down from the defaults of 100/20, sized for the original ~82-engine set).

### Logging

Both the `searxng` and `searxng-go` services set an explicit Docker `logging` driver (`json-file`, `max-size: 10m`, `max-file: 3`) in `docker-compose.yml`, so container logs rotate and are capped at ~30MB per service regardless of the host Docker daemon's own defaults.

## Available Make Commands

```bash
make help          # Show all available commands
make deps          # Download Go dependencies
make build         # Build the binary
make run           # Run locally (requires SEARXNG_URL)
make test          # Run tests
make docker-up     # Start Docker containers
make docker-down   # Stop Docker containers
make docker-logs   # View Docker logs
make install       # Install to /usr/local/bin
make build-all     # Build for multiple platforms
```

## Architecture

```
┌─────────────────────────┐
│   AI Assistant (Claude) │
└───────────┬─────────────┘
            │ MCP Protocol (stdio)
            │
┌───────────▼─────────────┐
│    MCP Server (Go)      │
│  ┌──────────────────┐   │
│  │  Search Client   │───┼──► SearXNG (Docker)
│  │  URL Reader      │───┼──► Internet
│  │  Cache (60s TTL) │   │
│  └──────────────────┘   │
└─────────────────────────┘
```

## Project Structure

```
mcp-searxng-claude-go/
├── main.go              # Application entry point
├── searxng.go          # SearXNG API client
├── urlreader.go        # URL fetching and HTML-to-Markdown conversion
├── cache.go            # In-memory caching with TTL
├── proxy.go            # HTTP proxy configuration
├── resources.go        # MCP resources (config, help)
├── Dockerfile          # Multi-stage Docker build
├── docker-compose.yml  # Service orchestration
├── Makefile            # Build automation
└── .env.example        # Environment template
```

