# websearch-mcp

Agent-oriented web information acquisition system that integrates multi-engine search and content fetching into a unified interface. Exposed via MCP server and LLM-friendly CLI, it enables structured, multi-source retrieval of web data for downstream LLM workflows.

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
websearch-cli fetch <url> [--method direct|cdp] [--cdp-mode connect|system|bundled] [--mode full|summary|title] [--no-cache]
```

Outputs LLM-friendly text to stdout. Exit code 0 on success, non-zero on failure.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8848` | Server listen port |
| `SEARCH_ENGINE` | `duckduckgo` | Default search engine (`duckduckgo` or `tavily`) |
| `TAVILY_API_KEY` | — | API key for Tavily search |
| `TAVILY_SEARCH_DEPTH` | `basic` | Tavily search depth: `basic`, `advanced`, `fast`, or `ultra-fast` |
| `TAVILY_MAX_RESULTS` | `7` | Max results per Tavily search (1-20) |
| `TAVILY_TOPIC` | `general` | Tavily search topic: `general`, `news`, or `finance` |
| `FETCH_METHOD` | `direct` | Default fetch method (`direct` or `cdp`) |
| `CHROME_DEBUG_ADDR` | `localhost:9222` | Chrome DevTools WebSocket address (used by `cdp` `connect` mode) |
| `CDP_MODE` | `connect` | Browser source for `cdp` method: `connect`, `system`, or `bundled` |
| `CHROME_BIN` | — | Override path to Chrome/Chromium binary (for `CDP_MODE=system`) |
| `CACHE_MAX_ENTRIES` | `128` | Max fetch cache entries |
| `CACHE_TTL` | `5m` | Cache time-to-live (e.g. `30s`, `5m`, `1h`) |
| `CACHE_MAX_ENTRY_SIZE` | `524288` | Max bytes per cached entry (512KB default) |

Priority: explicit request parameter > CLI flag > environment variable > default value.

All parameters validate their values. Passing an unknown value (e.g. `--method=foobar`, `engine: "saki"`) produces an error listing the valid choices. Omitting a parameter uses the default.

## Proxy Configuration

Go’s `net/http` uses `ProxyFromEnvironment` by default, which reads proxy settings from environment variables (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`). It does not automatically integrate with OS-level proxy settings (such as Windows Internet Options or macOS Network Preferences).

If you are behind a proxy, set these before running:

```bash
# Linux / macOS
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890

# Windows (PowerShell)
$env:HTTP_PROXY = "http://127.0.0.1:7890"
$env:HTTPS_PROXY = "http://127.0.0.1:7890"

# Windows (Command Prompt)
set HTTP_PROXY=http://127.0.0.1:7890
set HTTPS_PROXY=http://127.0.0.1:7890
```

Replace `127.0.0.1:7890` with your proxy address. Use `NO_PROXY` to exclude hosts (e.g. `NO_PROXY=localhost,127.0.0.1,::1,.local`).

For the `cdp` fetch method, the Chrome browser does **not** use `HTTP_PROXY` / `HTTPS_PROXY`. Instead, Chrome relies on:

- OS system proxy settings (Windows Internet Options, macOS Network Preferences)
- Or an explicit `--proxy-server` flag

None of the `CDP_MODE` options pass `--proxy-server` by default. In `connect` mode, add it when starting Chrome:

```bash
chrome --remote-debugging-port=9222 --proxy-server=http://127.0.0.1:7890
```

In `system` or `bundled` mode, Chrome may still work through the OS system proxy (e.g. Clash on Windows sets this automatically), but this is OS-dependent, not controlled by the tool.

## CDP Browser Modes

When `FETCH_METHOD=cdp`, the `CDP_MODE` env var controls how the Chrome/Chromium browser is acquired:

| Mode | Behavior | First Use | Cleanup |
|------|----------|-----------|---------|
| `connect` (default) | Connect to an already-running Chrome at `CHROME_DEBUG_ADDR` | None | Never kills the browser |
| `system` | Find and launch system-installed Chrome/Chromium/Edge | None; errors if not found (hint: use `bundled`) | Kills browser on exit |
| `bundled` | Auto-download rod's own Chromium | Downloads ~150MB | Kills browser on exit |

### `connect` mode

The default. Requires Chrome to be started with a debug port beforehand:

```bash
# Must use --user-data-dir if Chrome is already running, otherwise the flag is ignored
chrome --remote-debugging-port=9222 --user-data-dir=<temp-dir>
```

**Common pitfall:** Running `chrome --remote-debugging-port=9222` while Chrome is already open does **nothing** — the existing process silently ignores the flag. Always pass `--user-data-dir` to force a new instance, or use `system`/`bundled` mode instead.

### `system` mode

Finds your system browser automatically. Checks `CHROME_BIN` env var first, then looks for `google-chrome`, `chromium`, `chrome`, `msedge`, `brave` on `PATH`. If none are found, reports an error suggesting to switch to `CDP_MODE=bundled`.

### `bundled` mode

No manual setup required. On first run, rod downloads its own Chromium binary (~150MB). Subsequent runs reuse the cached binary. No system browser required.

## MCP Tools

### `search`

Search the web and return results with URL, title, and snippet.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |
| `engine` | string | no | `duckduckgo` or `tavily` (default: `SEARCH_ENGINE` env or `duckduckgo`) |

### `fetch_content`

Fetch a URL, convert HTML to readable text, and return content. The `mode` parameter controls how much content is returned: `"full"` for complete content, `"summary"` for a longer preview, `"title"` for a short preview only (default 900-char preview).

The `cdp` method renders JavaScript via Chrome DevTools Protocol. Use the `cdp_mode` argument to control browser acquisition: `connect` (default, requires Chrome pre-started at `CHROME_DEBUG_ADDR`), `system` (find and launch system browser), or `bundled` (auto-download Chromium). Use `direct` (plain HTTP, strips HTML) when Chrome is unavailable.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | URL of the page to fetch |
| `mode` | string | no | Content length — `"full"` (complete), `"summary"` (longer preview), `"title"` (short preview). Default: 900-char preview |
| `method` | string | no | `direct` or `cdp` (default: `FETCH_METHOD` env or `direct`). When `cdp`, use `cdp_mode` to choose the browser source. |
| `cdp_mode` | string | no | When `method=cdp`: `connect` (default, needs Chrome pre-started), `system` (find system browser), or `bundled` (auto-download Chromium). Defaults to `CDP_MODE` env var or `connect`. |
| `no_cache` | boolean | no | If `true`, bypass cache and force a fresh request (default: `false`) |

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
    "protocolVersion": "2025-06-18",
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
    "protocolVersion": "2025-06-18",
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

Request (direct):
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "fetch_content",
    "arguments": {
      "url": "https://example.com",
      "mode": "summary",
      "method": "direct",
      "no_cache": false
    }
  }
}
```

Request (cdp with bundled browser):
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "fetch_content",
    "arguments": {
      "url": "https://example.com",
      "mode": "full",
      "method": "cdp",
      "cdp_mode": "bundled",
      "no_cache": true
    }
  }
}
```
Note: `cdp` method renders JavaScript. Use `cdp_mode` to control browser acquisition: `connect` (default), `system`, or `bundled`. Omitting `cdp_mode` falls back to the `CDP_MODE` env var.

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
