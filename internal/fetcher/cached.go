package fetcher

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/iharee/websearch-mcp/internal/cache"
	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/model"
)

const (
	defaultPreviewChars = 900
	summaryPreviewChars = 1200
	titlePreviewChars   = 600
)

// CachedFetcher wraps a Provider with an LRU+TTL cache and content truncation.
type CachedFetcher struct {
	inner        Provider
	cache        *cache.Cache[*model.FetchResult]
	maxEntrySize int
}

// NewCachedFetcher creates a CachedFetcher wrapping the given provider.
func NewCachedFetcher(inner Provider) *CachedFetcher {
	return &CachedFetcher{
		inner: inner,
		cache: cache.New[*model.FetchResult](
			config.CacheMaxEntries(),
			config.CacheTTL(),
		),
		maxEntrySize: config.CacheMaxEntrySize(),
	}
}

// Warmup triggers any expensive one-time setup (e.g. launching a browser) outside the fetch timeout.
func (c *CachedFetcher) Warmup() {
	if w, ok := c.inner.(Warmupper); ok {
		w.Warmup()
	}
}

// Fetch returns page content, using the cache when available and noCache is false. Content is truncated according to mode.
func (c *CachedFetcher) Fetch(ctx context.Context, url string, mode string, noCache bool) (*model.FetchResult, error) {
	if !noCache {
		if cached, ok := c.cache.Get(url); ok {
			content, err := truncateByMode(cached.Content, mode)
			if err != nil {
				return nil, err
			}
			return &model.FetchResult{
				URL:     cached.URL,
				Title:   cached.Title,
				Content: content,
			}, nil
		}
	}

	full, err := c.inner.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}

	if !noCache && c.maxEntrySize > 0 && len(full.Content) <= c.maxEntrySize {
		c.cache.Put(url, full)
	}

	content, err := truncateByMode(full.Content, mode)
	if err != nil {
		return nil, err
	}
	return &model.FetchResult{
		URL:     full.URL,
		Title:   full.Title,
		Content: content,
	}, nil
}

func truncateByMode(fullText, mode string) (string, error) {
	lower := strings.ToLower(mode)
	switch {
	case lower == "":
		return previewText(fullText, defaultPreviewChars), nil
	case lower == "full":
		return fullText, nil
	case lower == "title":
		return previewText(fullText, titlePreviewChars), nil
	case lower == "summary" || lower == "summarize":
		return previewText(fullText, summaryPreviewChars), nil
	default:
		return "", fmt.Errorf("invalid mode: %q; valid modes are full, summary, title", mode)
	}
}

func previewText(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return strings.TrimSpace(string(runes[:maxChars])) + "..."
}

// Close releases underlying resources if the inner provider implements io.Closer.
func (c *CachedFetcher) Close() error {
	if closer, ok := c.inner.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
