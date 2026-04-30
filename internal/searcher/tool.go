package searcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/mcp"
	"github.com/iharee/websearch-mcp/internal/searcher/duckduckgo"
	"github.com/iharee/websearch-mcp/internal/searcher/tavily"
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

		provider, err := resolveProvider(args)
		if err != nil {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}

		results, err := provider.Search(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: fmt.Sprintf("No web search results matched the query %q.", query)}},
			}, nil
		}

		var buf strings.Builder
		fmt.Fprintf(&buf, "Search results for %q. Include a Sources section in the final answer.\n", query)
		for _, r := range results {
			fmt.Fprintf(&buf, "- [%s](%s)\n", r.Title, r.URL)
		}

		return &mcp.ToolCallResult{
			Content: []mcp.ContentItem{{Type: "text", Text: buf.String()}},
		}, nil
	}
}

func resolveProvider(args map[string]interface{}) (Provider, error) {
	engine := ""
	if e, ok := args["engine"].(string); ok {
		engine = strings.ToLower(strings.TrimSpace(e))
	}
	if engine == "" {
		engine = config.SearchEngine()
	}

	switch engine {
	case "duckduckgo":
		return duckduckgo.NewProvider(), nil
	case "tavily":
		return tavily.NewProvider(), nil
	default:
		return nil, fmt.Errorf("unknown search engine %q; valid engines: duckduckgo, tavily", engine)
	}
}
