package cdp

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type systemSource struct {
	mu       sync.Mutex
	launcher *launcher.Launcher
	binPath  string
}

func newSystemSource() *systemSource {
	return &systemSource{binPath: findSystemChrome()}
}

func findSystemChrome() string {
	if bin := os.Getenv("CHROME_BIN"); bin != "" {
		return bin
	}
	for _, name := range []string{"google-chrome", "chromium", "chrome", "msedge", "brave"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func (s *systemSource) Acquire() (*rod.Browser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.binPath == "" {
		return nil, fmt.Errorf("cdp: no system Chrome/Chromium found — set CHROME_BIN to the browser path, or switch to CDP_MODE=bundled to auto-download Chromium")
	}

	l := launcher.New().Headless(true).Leakless(true).Bin(s.binPath)

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("cdp: failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("cdp: connect to launched browser: %w", err)
	}

	s.launcher = l
	return browser, nil
}

func (s *systemSource) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.launcher != nil {
		s.launcher.Kill()
		s.launcher = nil
	}
}
