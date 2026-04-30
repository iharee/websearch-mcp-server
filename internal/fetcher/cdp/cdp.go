package cdp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/iharee/websearch-mcp/internal/model"
)

const (
	defaultAddr        = "localhost:9222"
	navigationTimeout  = 30 * time.Second
	networkIdleTimeout = time.Second

	defaultPreviewChars = 900
	summaryPreviewChars = 1200
	titlePreviewChars   = 600
)

// Provider implements fetcher.Provider using Chrome DevTools Protocol via go-rod.
type Provider struct {
	addr    string
	mu      sync.Mutex
	browser *rod.Browser
}

// NewProvider creates a CDP provider. It reads CHROME_DEBUG_ADDR from the environment; if unset, defaults to localhost:9222.
func NewProvider() *Provider {
	addr := os.Getenv("CHROME_DEBUG_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	return &Provider{addr: addr}
}

// connect establishes a WebSocket connection to the Chrome DevTools endpoint.
func (p *Provider) connect() error {
	u, err := launcher.ResolveURL(p.addr)
	if err != nil {
		return fmt.Errorf("cdp: cannot connect to Chrome at %s. Start Chrome with --remote-debugging-port=<port>, or use method=direct", p.addr)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("cdp: cannot connect to Chrome at %s. Start Chrome with --remote-debugging-port=<port>, or use method=direct", p.addr)
	}
	p.browser = browser
	return nil
}

// disconnect closes the browser connection and nils out the reference.
func (p *Provider) disconnect() {
	if p.browser != nil {
		p.browser.Close()
		p.browser = nil
	}
}

// Fetch navigates to the given URL in a fresh Chrome tab and returns the page title and body text.
func (p *Provider) Fetch(ctx context.Context, url string, prompt string) (*model.FetchResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Lazy connect on first use.
	if p.browser == nil {
		if err := p.connect(); err != nil {
			return nil, err
		}
	}

	// Open a fresh tab.
	page, err := p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		// Disconnect, reconnect, and retry once.
		p.disconnect()
		if connectErr := p.connect(); connectErr != nil {
			return nil, connectErr
		}
		page, err = p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return nil, fmt.Errorf("cdp: failed to open page: %w", err)
		}
	}
	defer page.Close()

	// Set user context and wrap with navigation timeout.
	page = page.Context(ctx).Timeout(navigationTimeout)

	// Navigate to the target URL.
	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("cdp: navigate: %w", err)
	}

	// Wait for the page load event.
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("cdp: wait load: %w", err)
	}

	// Wait for network to become idle.
	page.WaitRequestIdle(networkIdleTimeout, nil, nil, nil)()

	// Cancel the navigation timeout before metadata extraction so the timeout does not interfere with Info / Eval.
	page = page.CancelTimeout()

	// Extract page metadata (title, final URL).
	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("cdp: page info: %w", err)
	}

	// Extract body text via JavaScript evaluation.
	result, err := page.Eval("() => document.body?.innerText || ''")
	if err != nil {
		return nil, fmt.Errorf("cdp: eval body: %w", err)
	}

	fullContent := result.Value.Str()

	return &model.FetchResult{
		URL:     info.URL,
		Title:   info.Title,
		Content: selectContent(fullContent, prompt),
	}, nil
}

func selectContent(fullText, prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "full"):
		return fullText
	case strings.Contains(lower, "title"):
		return previewText(fullText, titlePreviewChars)
	case strings.Contains(lower, "summary") || strings.Contains(lower, "summarize"):
		return previewText(fullText, summaryPreviewChars)
	default:
		return previewText(fullText, defaultPreviewChars)
	}
}

func previewText(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "..."
}
