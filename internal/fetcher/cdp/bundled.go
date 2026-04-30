package cdp

import (
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type bundledSource struct {
	mu       sync.Mutex
	launcher *launcher.Launcher
}

func newBundledSource() *bundledSource {
	return &bundledSource{}
}

func (s *bundledSource) Acquire() (*rod.Browser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// launcher.New() auto-downloads rod's own Chromium if not present.
	l := launcher.New().Headless(true).Leakless(true)
	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("cdp: failed to launch bundled browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("cdp: connect to bundled browser: %w", err)
	}

	s.launcher = l
	return browser, nil
}

func (s *bundledSource) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.launcher != nil {
		s.launcher.Kill()
		s.launcher = nil
	}
}
