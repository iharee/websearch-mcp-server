package searcher

import (
	"context"

	"github.com/iharee/websearch-mcp-server/internal/model"
)

type Provider interface {
	Search(ctx context.Context, query string) ([]model.SearchResult, error)
}

type Parser interface {
	Parse(data []byte) ([]model.SearchResult, error)
}
