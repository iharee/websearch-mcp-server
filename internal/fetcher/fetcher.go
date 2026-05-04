package fetcher

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/fetcher/cdp"
	"github.com/iharee/websearch-mcp/internal/fetcher/direct"
	"github.com/iharee/websearch-mcp/internal/model"
)

var (
	directFetcher *CachedFetcher
	directOnce    sync.Once
	cdpFetchers   = make(map[string]*CachedFetcher)
	fetchersMu    sync.Mutex
)

var (
	closers   []io.Closer
	closersMu sync.Mutex
)

// Shutdown closes all registered closable fetchers. Call on process exit.
// Safe to call multiple times; subsequent calls are no-ops.
func Shutdown() {
	closersMu.Lock()
	defer closersMu.Unlock()
	for _, c := range closers {
		_ = c.Close()
	}
	closers = nil
}

func registerCloser(c io.Closer) {
	closersMu.Lock()
	defer closersMu.Unlock()
	closers = append(closers, c)
}

func newDirectFetcher() *CachedFetcher {
	directOnce.Do(func() {
		directFetcher = NewCachedFetcher(direct.NewProvider())
	})
	return directFetcher
}

func getCdpFetcher(cdpMode string) *CachedFetcher {
	fetchersMu.Lock()
	if f, ok := cdpFetchers[cdpMode]; ok {
		fetchersMu.Unlock()
		return f
	}
	provider := cdp.NewProvider(cdpMode)
	f := NewCachedFetcher(provider)
	cdpFetchers[cdpMode] = f
	fetchersMu.Unlock()

	registerCloser(f) // outside fetchersMu to avoid lock ordering with closersMu
	return f
}

// Resolve returns the CachedFetcher for the given method and cdpMode, validating both.
func Resolve(method, cdpMode string) (*CachedFetcher, error) {
	if method == "" {
		method = config.FetchMethod()
	}
	if method != "direct" && method != "cdp" {
		return nil, fmt.Errorf("unknown fetch method %q; valid methods: direct, cdp", method)
	}

	if method == "cdp" {
		if cdpMode == "" {
			cdpMode = config.CdpMode()
		}
		if cdpMode != "connect" && cdpMode != "system" && cdpMode != "bundled" {
			return nil, fmt.Errorf("unknown CDP_MODE %q; valid modes: connect, system, bundled", cdpMode)
		}
		return getCdpFetcher(cdpMode), nil
	}

	return newDirectFetcher(), nil
}

// FormatResult formats a fetch result for display.
func FormatResult(content *model.FetchResult) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "Title: %s\n", content.Title)
	fmt.Fprintf(&buf, "URL: %s\n", content.URL)
	if content.Content != "" {
		fmt.Fprintf(&buf, "\n%s", content.Content)
	}
	return buf.String()
}
