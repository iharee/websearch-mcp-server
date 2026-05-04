package cdp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/iharee/websearch-mcp/internal/model"
)

const (
	navigationTimeout  = 30 * time.Second
	networkIdleTimeout = time.Second
)

// Provider implements fetcher.Provider using Chrome DevTools Protocol via go-rod.
type Provider struct {
	source  BrowserSource
	mu      sync.Mutex
	browser *rod.Browser
}

func NewProvider(mode string) *Provider {
	return &Provider{source: newSource(mode)}
}

// Warmup triggers browser acquisition eagerly so it does not consume the fetch timeout.
func (p *Provider) Warmup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.browser == nil {
		_ = p.connect() // errors surface on the next Fetch call
	}
}

func newSource(mode string) BrowserSource {
	switch strings.ToLower(mode) {
	case "system":
		return newSystemSource()
	case "bundled":
		return newBundledSource()
	default:
		return newConnectSource()
	}
}

func (p *Provider) connect() error {
	browser, err := p.source.Acquire()
	if err != nil {
		return err
	}
	p.browser = browser
	return nil
}

func (p *Provider) disconnect() {
	if p.browser != nil {
		p.browser.Close()
		p.browser = nil
	}
	p.source.Release()
}

// Close releases the browser. Safe to call multiple times.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.disconnect()
	return nil
}

func (p *Provider) Fetch(ctx context.Context, url string) (*model.FetchResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.browser == nil {
		if err := p.connect(); err != nil {
			return nil, err
		}
	}

	page, err := p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
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

	page = page.Context(ctx).Timeout(navigationTimeout)

	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("cdp: navigate: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("cdp: wait load: %w", err)
	}

	page.WaitRequestIdle(networkIdleTimeout, nil, nil, nil)()

	page = page.CancelTimeout()

	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("cdp: page info: %w", err)
	}

	result, err := page.Eval("() => document.body?.innerText || ''")
	if err != nil {
		return nil, fmt.Errorf("cdp: eval body: %w", err)
	}

	return &model.FetchResult{
		URL:     info.URL,
		Title:   info.Title,
		Content: result.Value.Str(),
	}, nil
}
