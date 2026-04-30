package cdp

import (
	"fmt"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const defaultAddr = "localhost:9222"

type connectSource struct {
	addr string
}

func newConnectSource() *connectSource {
	addr := os.Getenv("CHROME_DEBUG_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	return &connectSource{addr: addr}
}

func (s *connectSource) Acquire() (*rod.Browser, error) {
	u, err := launcher.ResolveURL(s.addr)
	if err != nil {
		return nil, fmt.Errorf("cdp: cannot connect to Chrome at %s — is Chrome running with --remote-debugging-port? "+
			"Try setting CDP_MODE=system or CDP_MODE=bundled to let the tool manage the browser for you. "+
			"See Proxy Configuration in README if you are behind a proxy.", s.addr)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("cdp: cannot connect to Chrome at %s — is Chrome running with --remote-debugging-port? "+
			"Try setting CDP_MODE=system or CDP_MODE=bundled.", s.addr)
	}
	return browser, nil
}

func (s *connectSource) Release() {
	// Never kill an externally-managed browser; the caller only closes the rod handle.
}
