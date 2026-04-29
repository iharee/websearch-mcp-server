# websearch-mcp-server

![Status](https://img.shields.io/badge/status-alpha-orange)
![WIP](https://img.shields.io/badge/🚧-WIP-yellow)

Advanced web search tools and CLI for agents via MCP.

## Features

- **Multi-engine web search** — DuckDuckGo and Tavily, selectable per query
- **Content fetching** — fetch full page content by URL (browser-style)

## Quick Start

```bash
go build -o websearch-mcp-server ./cmd/server/
./websearch-mcp-server
```

Server listens on port `8848` (configurable via `PORT` env var).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8848` | Server listen port |
| `SEARCH_ENGINE` | `duckduckgo` | Default search engine (`duckduckgo` or `tavily`) |
| `TAVILY_API_KEY` | — | API key for Tavily search |

## MCP Tools

### `search`

Search the web and return results with URL, title, and snippet.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |
| `engine` | string | no | `duckduckgo` or `tavily` (default: `SEARCH_ENGINE` env or `duckduckgo`) |

### `fetch_content`

Fetch the full text content of a web page.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | URL of the page to fetch |

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
      "name": "websearch-mcp-server",
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
        "text": "[{\"url\":\"\",\"title\":\"Tenma Saki\",\"snippet\":\"\"},{\"url\":\"\",\"title\":\"SEGA copyright\",\"snippet\":\"\"}]"
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
      "url": "https://example.com"
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
        "text": "{\"url\":\"https://example.com\",\"title\":\"Example Domain\",\"content\":\"...\"}"
      }
    ]
  }
}
```

## CLI

For AI agents that prefer direct invocation without MCP protocol overhead:

```bash
go build -o websearch-cli ./cmd/cli/

# Search
./websearch-cli search --query "golang best practices" --engine duckduckgo

# Fetch
./websearch-cli fetch --url "https://example.com"
```

Outputs JSON to stdout. Exit code 0 on success, non-zero on failure.
