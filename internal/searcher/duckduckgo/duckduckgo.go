package duckduckgo

import (
	"context"

	"github.com/iharee/websearch-mcp-server/internal/model"
)

type Provider struct {
	parser *Parser
}

func NewProvider() *Provider {
	return &Provider{parser: &Parser{}}
}

func (p *Provider) Search(_ context.Context, query string) ([]model.SearchResult, error) {
	return p.parser.Parse(nil)
}

type Parser struct{}

func (Parser) Parse(_ []byte) ([]model.SearchResult, error) {
	return []model.SearchResult{
		{URL: "https://example.com/ddg/1", Title: "DDG Result One", Snippet: "A fixed DuckDuckGo search result snippet."},
		{URL: "https://example.com/ddg/2", Title: "DDG Result Two", Snippet: "Another fixed DuckDuckGo snippet."},
	}, nil
}
