package cdp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

const testURL = "https://iharee.github.io/2026/03/22/mathematical_principles_of_transformer/"

func TestNewProvider(t *testing.T) {
	t.Run("default_connect", func(t *testing.T) {
		os.Unsetenv("CDP_MODE")
		p := NewProvider()
		_, ok := p.source.(*connectSource)
		if !ok {
			t.Errorf("default source type = %T, want *connectSource", p.source)
		}
	})

	t.Run("system_mode", func(t *testing.T) {
		os.Setenv("CDP_MODE", "system")
		defer os.Unsetenv("CDP_MODE")
		p := NewProvider()
		_, ok := p.source.(*systemSource)
		if !ok {
			t.Errorf("source type = %T, want *systemSource", p.source)
		}
	})

	t.Run("bundled_mode", func(t *testing.T) {
		os.Setenv("CDP_MODE", "bundled")
		defer os.Unsetenv("CDP_MODE")
		p := NewProvider()
		_, ok := p.source.(*bundledSource)
		if !ok {
			t.Errorf("source type = %T, want *bundledSource", p.source)
		}
	})

	t.Run("unknown_mode_falls_back_to_connect", func(t *testing.T) {
		os.Setenv("CDP_MODE", "unknown")
		defer os.Unsetenv("CDP_MODE")
		p := NewProvider()
		_, ok := p.source.(*connectSource)
		if !ok {
			t.Errorf("source type = %T, want *connectSource (fallback)", p.source)
		}
	})
}

func TestConnectSourceAddr(t *testing.T) {
	t.Run("default_addr", func(t *testing.T) {
		os.Unsetenv("CHROME_DEBUG_ADDR")
		s := newConnectSource()
		if s.addr != "localhost:9222" {
			t.Errorf("default addr = %q, want localhost:9222", s.addr)
		}
	})

	t.Run("custom_addr", func(t *testing.T) {
		os.Setenv("CHROME_DEBUG_ADDR", "127.0.0.1:9999")
		defer os.Unsetenv("CHROME_DEBUG_ADDR")
		s := newConnectSource()
		if s.addr != "127.0.0.1:9999" {
			t.Errorf("addr = %q, want 127.0.0.1:9999", s.addr)
		}
	})
}

func TestFetchNoChrome(t *testing.T) {
	os.Setenv("CHROME_DEBUG_ADDR", "localhost:19999")
	defer os.Unsetenv("CHROME_DEBUG_ADDR")

	p := NewProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.Fetch(ctx, "https://example.com")
	if err == nil {
		t.Fatal("expected error when Chrome is not running")
	}
	if !strings.Contains(err.Error(), "cannot connect to Chrome") {
		t.Errorf("error should mention cannot connect, got: %v", err)
	}
}

func TestFetchIntegration(t *testing.T) {
	if os.Getenv("CHROME_DEBUG_ADDR") == "" {
		t.Skip("CHROME_DEBUG_ADDR not set, skipping integration test (start Chrome with --remote-debugging-port=9222)")
	}

	p := NewProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := p.Fetch(ctx, testURL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if result.URL == "" {
		t.Error("URL is empty")
	}
	if result.Title == "" {
		t.Error("Title is empty")
	}
	if result.Content == "" {
		t.Error("Content is empty")
	}

	if strings.Contains(result.Content, "<") && strings.Contains(result.Content, ">") {
		t.Error("Content appears to contain HTML tags — expected plain text from innerText")
	}

	t.Logf("URL: %s", result.URL)
	t.Logf("Title: %s", result.Title)
	t.Logf("Content length: %d chars", len(result.Content))
}

func TestFetchRedirect(t *testing.T) {
	if os.Getenv("CHROME_DEBUG_ADDR") == "" {
		t.Skip("CHROME_DEBUG_ADDR not set, skipping integration test")
	}

	p := NewProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := p.Fetch(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !strings.Contains(result.URL, "https://") && !strings.Contains(result.URL, "example.com") {
		t.Errorf("expected post-redirect URL, got: %s", result.URL)
	}

	t.Logf("Final URL: %s", result.URL)
}

func TestFetchContextCancellation(t *testing.T) {
	if os.Getenv("CHROME_DEBUG_ADDR") == "" {
		t.Skip("CHROME_DEBUG_ADDR not set, skipping integration test")
	}

	p := NewProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.Fetch(ctx, "https://example.com")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
