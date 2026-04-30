package searcher

import (
	"context"
	"strings"

	"github.com/iharee/websearch-mcp/internal/mcp"
)

func ToolDefinition() mcp.Tool {
	return mcp.Tool{
		Name:        "search",
		Description: "Search the web and return a list of results with URL, title, and snippet. Use the 'engine' parameter to select duckduckgo or tavily.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.JSONSchema{
				"query": {
					Type:        "string",
					Description: "Search query",
				},
				"engine": {
					Type:        "string",
					Description: "Search engine: duckduckgo or tavily (case-insensitive). Defaults to SEARCH_ENGINE env var or duckduckgo.",
				},
			},
			Required: []string{"query"},
		},
	}
}

func Handler() mcp.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.ToolCallResult, error) {
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: "missing required argument: query"}},
				IsError: true,
			}, nil
		}

		engine, _ := args["engine"].(string)
		provider, err := Resolve(strings.ToLower(strings.TrimSpace(engine)))
		if err != nil {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}

		results, err := provider.Search(ctx, query)
		if err != nil {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: "search failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.ToolCallResult{
			Content: []mcp.ContentItem{{Type: "text", Text: FormatResults(query, results)}},
		}, nil
	}
}
