package searcher

import (
	"fmt"
	"strings"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/model"
	"github.com/iharee/websearch-mcp/internal/searcher/duckduckgo"
	"github.com/iharee/websearch-mcp/internal/searcher/tavily"
)

// Resolve returns the Provider for the given engine, validating it.
func Resolve(engine string) (Provider, error) {
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

// FormatResults formats search results for display.
func FormatResults(query string, results []model.SearchResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No web search results matched the query %q.\n", query)
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "Search results for %q. Include a Sources section in the final answer.\n", query)
	for _, r := range results {
		fmt.Fprintf(&buf, "- [%s](%s)\n", r.Title, r.URL)
	}
	return buf.String()
}
