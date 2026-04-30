package fetcher

import (
	"context"
	"strings"

	"github.com/iharee/websearch-mcp/internal/mcp"
)

func ToolDefinition() mcp.Tool {
	return mcp.Tool{
		Name:        "fetch_content",
		Description: "Fetch a URL, convert HTML to readable text, and return content based on the prompt intent. Use 'method' parameter to select direct or cdp.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.JSONSchema{
				"url": {
					Type:        "string",
					Description: "URL of the page to fetch",
				},
				"mode": {
					Type:        "string",
					Description: "Content length mode: 'full' (complete), 'summary' (longer preview), 'title' (short preview). Defaults to a 900-char preview.",
				},
				"method": {
					Type:        "string",
					Description: "Fetch method: direct or cdp (case-insensitive). Defaults to FETCH_METHOD env var or direct.",
				},
				"cdp_mode": {
					Type:        "string",
					Description: "Browser source when method=cdp: connect (default, needs Chrome pre-started), system (find system browser), or bundled (auto-download Chromium). Defaults to CDP_MODE env var or connect.",
				},
				"no_cache": {
					Type:        "boolean",
					Description: "If true, bypass the cache and force a fresh fetch. Defaults to false.",
				},
			},
			Required: []string{"url"},
		},
	}
}

func Handler() mcp.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.ToolCallResult, error) {
		url, ok := args["url"].(string)
		if !ok || url == "" {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: "missing required argument: url"}},
				IsError: true,
			}, nil
		}

		method, _ := args["method"].(string)
		cdpMode, _ := args["cdp_mode"].(string)

		fetcher, err := Resolve(strings.ToLower(strings.TrimSpace(method)), strings.ToLower(strings.TrimSpace(cdpMode)))
		if err != nil {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		fetcher.Warmup()

		mode := ""
		if m, ok := args["mode"].(string); ok {
			mode = strings.TrimSpace(m)
		}

		noCache := false
		if nc, ok := args["no_cache"].(bool); ok {
			noCache = nc
		}

		result, err := fetcher.Fetch(ctx, url, mode, noCache)
		if err != nil {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: "fetch failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.ToolCallResult{
			Content: []mcp.ContentItem{{Type: "text", Text: FormatResult(result)}},
		}, nil
	}
}
