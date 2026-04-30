# websearch-mcp

![Status](https://img.shields.io/badge/status-alpha-orange)
![WIP](https://img.shields.io/badge/🚧-WIP-yellow)

Advanced web search tools designed for agents. Available as both MCP server and LLM-friendly CLI, supporting multiple search engines and various ways to gather internet information.

## Features

- **Multi-engine web search** — DuckDuckGo and Tavily, selectable per query
- **Content fetching** — fetch full page content by URL (direct HTTP or CDP-based)
- **Dual interface** — MCP server for protocol-based integration, CLI for direct invocation

## Quick Start

### MCP Server

```bash
go build -o websearch-mcp ./cmd/server/
./websearch-mcp
```

Server listens on port `8848` (configurable via `PORT` env var or `--port` flag).

### CLI

```bash
go build -o websearch-cli ./cmd/cli/
```

```bash
# Search
websearch-cli search <query> [--engine duckduckgo|tavily]

# Fetch
websearch-cli fetch <url> [--method direct|cdp] [--mode full|summary|title]
```

Outputs LLM-friendly text to stdout. Exit code 0 on success, non-zero on failure.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8848` | Server listen port |
| `SEARCH_ENGINE` | `duckduckgo` | Default search engine (`duckduckgo` or `tavily`) |
| `TAVILY_API_KEY` | — | API key for Tavily search |
| `FETCH_METHOD` | `direct` | Default fetch method (`direct` or `cdp`) |
| `CHROME_DEBUG_ADDR` | `localhost:9222` | Chrome DevTools WebSocket address (used by `cdp` method) |

Priority: explicit request parameter > CLI flag > environment variable > default value.

## MCP Tools

### `search`

Search the web and return results with URL, title, and snippet.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |
| `engine` | string | no | `duckduckgo` or `tavily` (default: `SEARCH_ENGINE` env or `duckduckgo`) |

### `fetch_content`

Fetch a URL, convert HTML to readable text, and return content. The `mode` parameter controls how much content is returned: `"full"` for complete content, `"summary"` for a longer preview, `"title"` for a short preview only (default 900-char preview).

The `cdp` method renders JavaScript via Chrome DevTools Protocol. It requires Chrome running with `--remote-debugging-port=<port>` (default 9222). Use `direct` (plain HTTP, strips HTML) when Chrome is unavailable.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | URL of the page to fetch |
| `mode` | string | no | Content length — `"full"` (complete), `"summary"` (longer preview), `"title"` (short preview). Default: 900-char preview |
| `method` | string | no | `direct` or `cdp` (default: `FETCH_METHOD` env or `direct`). `cdp` renders JavaScript but requires Chrome running with `--remote-debugging-port`. |

## MCP Protocol Examples

The server speaks JSON-RPC 2.0 at `POST /mcp`. Initialize first to get a session.

### Initialize

Request:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2026-04-29",
    "capabilities": {},
    "clientInfo": {
      "name": "test",
      "version": "1.0"
    }
  }
}
```

Response (`Mcp-Session-Id` also in header):
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2026-04-29",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "websearch-mcp",
      "version": "0.1.0"
    }
  }
}
```

### `search`

Request:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search",
    "arguments": {
      "query": "tenmasaki",
      "engine": "duckduckgo"
    }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Search results for \"tenmasaki\". Include a Sources section in the final answer.\n- [Tenma Saki](https://example_1.com/)\n- [SEGA copyright](https://example_2.com/)"
      }
    ]
  }
}
```

### `fetch_content`

Request:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "fetch_content",
    "arguments": {
      "url": "https://example.com",
      "method": "direct"
    }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Title: Example Domain\nURL: https://example.com\n\nThis domain is for use in illustrative examples in documents..."
      }
    ]
  }
}
```
