package fetcher

import (
	"context"

	"github.com/iharee/websearch-mcp-server/internal/model"
)

type Provider interface {
	Fetch(ctx context.Context, url string, prompt string) (*model.FetchResult, error)
}
