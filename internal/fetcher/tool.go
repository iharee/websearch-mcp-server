package fetcher

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/fetcher/cdp"
	"github.com/iharee/websearch-mcp/internal/fetcher/direct"
	"github.com/iharee/websearch-mcp/internal/mcp"
)

var (
	directFetcher  *CachedFetcher
	cdpFetchers   = make(map[string]*CachedFetcher)
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
	// Temporarily set CDP_MODE so newSource() picks the right mode.
	// CDP_MODE is read once at Provider creation time.
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

func ToolDefinition() mcp.Tool {
	return mcp.Tool{
		Name:        "fetch_content",
		Description: "Fetch a URL, convert HTML to readable text, and return content based on the prompt intent. Use 'method' parameter to select direct or cdp.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.JSONSchema{
				"url": {
					Type:        "string",
					Description: "URL of the page to fetch",
				},
				"mode": {
					Type:        "string",
					Description: "Content length mode: 'full' (complete), 'summary' (longer preview), 'title' (short preview). Defaults to a 900-char preview.",
				},
				"method": {
					Type:        "string",
					Description: "Fetch method: direct or cdp (case-insensitive). Defaults to FETCH_METHOD env var or direct.",
				},
				"cdp_mode": {
					Type:        "string",
					Description: "Browser source when method=cdp: connect (default, needs Chrome pre-started), system (find system browser), or bundled (auto-download Chromium). Defaults to CDP_MODE env var or connect.",
				},
				"no_cache": {
					Type:        "boolean",
					Description: "If true, bypass the cache and force a fresh fetch. Defaults to false.",
				},
			},
			Required: []string{"url"},
		},
	}
}

func Handler() mcp.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.ToolCallResult, error) {
		url, ok := args["url"].(string)
		if !ok || url == "" {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: "missing required argument: url"}},
				IsError: true,
			}, nil
		}

		mode := ""
		if m, ok := args["mode"].(string); ok {
			mode = strings.TrimSpace(m)
		}

		noCache := false
		if nc, ok := args["no_cache"].(bool); ok {
			noCache = nc
		}

		fetcher, err := resolveFetcher(args)
		if err != nil {
			return &mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		fetcher.Warmup()

		content, err := fetcher.Fetch(ctx, url, mode, noCache)
		if err != nil {
			return nil, fmt.Errorf("fetch failed: %w", err)
		}

		var buf strings.Builder
		fmt.Fprintf(&buf, "Title: %s\n", content.Title)
		fmt.Fprintf(&buf, "URL: %s\n", content.URL)
		if content.Content != "" {
			fmt.Fprintf(&buf, "\n%s", content.Content)
		}

		return &mcp.ToolCallResult{
			Content: []mcp.ContentItem{{Type: "text", Text: buf.String()}},
		}, nil
	}
}

func resolveFetcher(args map[string]interface{}) (*CachedFetcher, error) {
	method := ""
	if m, ok := args["method"].(string); ok {
		method = strings.ToLower(strings.TrimSpace(m))
	}
	if method == "" {
		method = config.FetchMethod()
	}
	if method != "direct" && method != "cdp" {
		return nil, fmt.Errorf("unknown fetch method %q; valid methods: direct, cdp", method)
	}

	if method == "cdp" {
		cdpMode := ""
		if m, ok := args["cdp_mode"].(string); ok {
			cdpMode = strings.ToLower(strings.TrimSpace(m))
		}
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
