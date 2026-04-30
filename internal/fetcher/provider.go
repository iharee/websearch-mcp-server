package fetcher

import (
	"context"

	"github.com/iharee/websearch-mcp/internal/model"
)

// Provider fetches a URL and returns the full page content. Caching and content truncation are handled by the caller.
type Provider interface {
	Fetch(ctx context.Context, url string) (*model.FetchResult, error)
}

// Warmupper is an optional interface for providers that need pre-warming (e.g. launching a browser).
type Warmupper interface {
	Warmup()
}
