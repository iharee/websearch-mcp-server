package searcher

import (
	"context"

	"github.com/iharee/websearch-mcp-server/internal/model"
)

type MockProvider struct{}

func (MockProvider) Search(_ context.Context, query string) ([]model.SearchResult, error) {
	return []model.SearchResult{
		{
			URL:     "https://en.wikipedia.org/wiki/" + query,
			Title:   query + " - Wikipedia",
			Snippet: "This is a mock search result for \"" + query + "\". Replace with a real search provider.",
		},
		{
			URL:     "https://example.com/results?q=" + query,
			Title:   "Results for " + query,
			Snippet: "Another mock result. Implement SearchProvider to connect to a real search API.",
		},
	}, nil
}
