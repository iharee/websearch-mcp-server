package searcher

import (
	"context"

	"github.com/iharee/websearch-mcp/internal/model"
)

type Provider interface {
	Search(ctx context.Context, query string) ([]model.SearchResult, error)
}
