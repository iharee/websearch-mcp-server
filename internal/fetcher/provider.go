package fetcher

import (
	"context"

	"github.com/iharee/websearch-mcp/internal/model"
)

type Provider interface {
	Fetch(ctx context.Context, url string, mode string) (*model.FetchResult, error)
}
