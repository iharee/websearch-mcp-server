package fetcher

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/fetcher/cdp"
	"github.com/iharee/websearch-mcp/internal/fetcher/direct"
	"github.com/iharee/websearch-mcp/internal/model"
)

var (
	directFetcher  *CachedFetcher
	cdpFetchers    = make(map[string]*CachedFetcher)
	fetchersMu     sync.Mutex
	fetchersInited bool
)

func initFetchers() {
	fetchersMu.Lock()
	defer fetchersMu.Unlock()
	if !fetchersInited {
		directFetcher = NewCachedFetcher(direct.NewProvider())
		fetchersInited = true
	}
}

func getCdpFetcher(cdpMode string) *CachedFetcher {
	fetchersMu.Lock()
	defer fetchersMu.Unlock()
	if f, ok := cdpFetchers[cdpMode]; ok {
		return f
	}
	prev := setEnv("CDP_MODE", cdpMode)
	f := NewCachedFetcher(cdp.NewProvider())
	setEnv("CDP_MODE", prev)
	cdpFetchers[cdpMode] = f
	return f
}

func setEnv(key, value string) string {
	prev := os.Getenv(key)
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
	return prev
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

	initFetchers()
	return directFetcher, nil
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
