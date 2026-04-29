package fetcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/iharee/websearch-mcp-server/internal/config"
	"github.com/iharee/websearch-mcp-server/internal/fetcher/browser"
	"github.com/iharee/websearch-mcp-server/internal/fetcher/direct"
	"github.com/iharee/websearch-mcp-server/internal/mcp"
)

func ToolDefinition() mcp.Tool {
	return mcp.Tool{
		Name:        "fetch_content",
		Description: "Fetch a URL, convert HTML to readable text, and return content based on the prompt intent. Use 'method' parameter to select direct or browser.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.JSONSchema{
				"url": {
					Type:        "string",
					Description: "URL of the page to fetch",
				},
				"prompt": {
					Type:        "string",
					Description: "What you want to extract from the page. Use 'title' for the page title, 'summary' for a longer preview, or describe what you're looking for to get full content.",
				},
				"method": {
					Type:        "string",
					Description: "Fetch method: direct or browser (case-insensitive). Defaults to FETCH_METHOD env var or browser.",
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

		prompt := ""
		if p, ok := args["prompt"].(string); ok {
			prompt = strings.TrimSpace(p)
		}

		provider := resolveProvider(args)

		content, err := provider.Fetch(ctx, url, prompt)
		if err != nil {
			return nil, fmt.Errorf("fetch failed: %w", err)
		}

		var buf strings.Builder
		fmt.Fprintf(&buf, "Title: %s\n", content.Title)
		fmt.Fprintf(&buf, "URL: %s\n", content.URL)
		if content.Content != "" {
			fmt.Fprintf(&buf, "\n%s", content.Content)
		}

		return &mcp.ToolCallResult{
			Content: []mcp.ContentItem{{Type: "text", Text: buf.String()}},
		}, nil
	}
}

func resolveProvider(args map[string]interface{}) Provider {
	method := ""
	if m, ok := args["method"].(string); ok {
		method = strings.ToLower(strings.TrimSpace(m))
	}
	if method == "" {
		method = config.FetchMethod()
	}

	switch method {
	case "browser":
		return browser.NewProvider()
	default:
		return direct.NewProvider()
	}
}
