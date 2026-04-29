package cdp

import (
	"context"
	"fmt"

	"github.com/iharee/websearch-mcp-server/internal/model"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Fetch(_ context.Context, url string, prompt string) (*model.FetchResult, error) {
	return nil, fmt.Errorf("cdp fetch: not implemented yet")
}
