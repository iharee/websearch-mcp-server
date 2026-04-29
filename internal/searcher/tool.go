package searcher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/iharee/websearch-mcp-server/internal/searcher/duckduckgo"
	"github.com/iharee/websearch-mcp-server/internal/searcher/tavily"
	"strings"

	"github.com/iharee/websearch-mcp-server/internal/config"
	"github.com/iharee/websearch-mcp-server/internal/mcp"
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

		provider := resolveProvider(args)

		results, err := provider.Search(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		data, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("marshal results: %w", err)
		}

		return &mcp.ToolCallResult{
			Content: []mcp.ContentItem{{Type: "text", Text: string(data)}},
		}, nil
	}
}

func resolveProvider(args map[string]interface{}) Provider {
	engine := ""
	if e, ok := args["engine"].(string); ok {
		engine = strings.ToLower(strings.TrimSpace(e))
	}
	if engine == "" {
		engine = config.SearchEngine()
	}

	switch engine {
	case "tavily":
		return tavily.NewProvider()
	default:
		return duckduckgo.NewProvider()
	}
}
