package duckduckgo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const ddgHTML = `<!DOCTYPE html>
<html>
<head><title>test at DuckDuckGo</title></head>
<body>
<div class="results">
  <div class="result">
	<a rel="nofollow" class="result__a" href="https://go.dev/doc/effective_go">Effective Go - The Go Programming Language</a>
	<div class="result__snippet">Go is a general-purpose language designed with systems programming in mind.</div>
  </div>
  <div class="result">
	<a rel="nofollow" class="result__a" href="/l/?uddg=https%3A%2F%2Fgithub.com%2Fgolang%2Fgo%2Fwiki%2FCodeReviewComments">Code Review Comments - GitHub Wiki</a>
	<div class="result__snippet">This page collects common comments made during reviews of Go code.</div>
  </div>
  <div class="result">
	<a rel="nofollow" class="result__a" href="//example.com/relative-protocol">Protocol-relative link</a>
	<div class="result__snippet">A link with a protocol-relative URL.</div>
  </div>
  <div class="result">
	<a rel="nofollow" class="result__a" href="https://go.dev/doc/effective_go">Duplicate - Effective Go</a>
	<div class="result__snippet">This is a duplicate entry to test deduplication.</div>
  </div>
</div>
</body>
</html>`

const fallbackHTML = `<!DOCTYPE html>
<html>
<head><title>No results found</title></head>
<body>
<div class="no-results">Sorry, no results.</div>
<a href="https://example.com/page1">Example Page 1</a>
<a href="https://example.com/page2">Example Page 2</a>
<a href="javascript:void(0)">Javascript Link</a>
</div>
</body>
</html>`

func TestExtractSearchHits(t *testing.T) {
	hits := extractSearchHits(ddgHTML)

	if len(hits) == 0 {
		t.Fatal("expected search hits, got none")
	}

	// First hit: direct URL
	if hits[0].URL != "https://go.dev/doc/effective_go" {
		t.Errorf("hit[0].URL = %q, want %q", hits[0].URL, "https://go.dev/doc/effective_go")
	}
	if hits[0].Title != "Effective Go - The Go Programming Language" {
		t.Errorf("hit[0].Title = %q, want %q", hits[0].Title, "Effective Go - The Go Programming Language")
	}

	// Second hit: DDG redirect URL
	if hits[1].URL != "https://github.com/golang/go/wiki/CodeReviewComments" {
		t.Errorf("hit[1].URL = %q, want github URL", hits[1].URL)
	}
	if hits[1].Title != "Code Review Comments - GitHub Wiki" {
		t.Errorf("hit[1].Title = %q, want %q", hits[1].Title, "Code Review Comments - GitHub Wiki")
	}

	// Third hit: protocol-relative link
	if hits[2].URL != "https://example.com/relative-protocol" {
		t.Errorf("hit[2].URL = %q, want %q", hits[2].URL, "https://example.com/relative-protocol")
	}
}

func TestExtractGenericLinks(t *testing.T) {
	hits := extractGenericLinks(fallbackHTML)

	if len(hits) == 0 {
		t.Fatal("expected generic link hits, got none")
	}

	if hits[0].URL != "https://example.com/page1" {
		t.Errorf("hit[0].URL = %q, want %q", hits[0].URL, "https://example.com/page1")
	}
	if hits[1].URL != "https://example.com/page2" {
		t.Errorf("hit[1].URL = %q, want %q", hits[1].URL, "https://example.com/page2")
	}

	// Javascript link should be filtered out
	for _, h := range hits {
		if h.URL == "javascript:void(0)" {
			t.Error("javascript link should not appear in results")
		}
	}
}

func TestDedupeHits(t *testing.T) {
	hits := extractSearchHits(ddgHTML)
	hits = append(hits, extractGenericLinks(fallbackHTML)...)

	dedupeHits(&hits)

	// Should not have duplicate "https://go.dev/doc/effective_go"
	count := 0
	for _, h := range hits {
		if h.URL == "https://go.dev/doc/effective_go" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("got %d duplicates of go.dev/effective_go, expected 1", count)
	}
}

func TestSearchFullPipeline(t *testing.T) {
	// Simulates the full DuckDuckGo HTML response through the real Search() flow.
	// Verifies: HTTP → body read → extractSearchHits → fallback skipped → dedupe → truncate.
	const respHTML = `<!DOCTYPE html>
<html>
<head><title>test at DuckDuckGo</title></head>
<body>
<div class="results">
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://go.dev/doc/effective_go">Effective Go</a>
    <div class="result__snippet">Go is a general-purpose language.</div>
  </div>
  <div class="result">
    <a rel="nofollow" class="result__a" href="/l/?uddg=https%3A%2F%2Fgithub.com%2Fgolang%2Fgo%2Fwiki%2FCodeReviewComments">Code Review Comments</a>
    <div class="result__snippet">Common comments made during reviews of Go code.</div>
  </div>
  <div class="result">
    <a rel="nofollow" class="result__a" href="//example.com/protocol-relative">Protocol-relative link</a>
    <div class="result__snippet">A link with a protocol-relative URL.</div>
  </div>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://go.dev/doc/effective_go">Duplicate Effective Go</a>
    <div class="result__snippet">This is a duplicate.</div>
  </div>
</div>
</body>
</html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q != "golang" {
			t.Errorf("unexpected query: %q", q)
		}
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(respHTML))
	}))
	defer srv.Close()

	p := &Provider{
		client:    srv.Client(),
		searchURL: srv.URL,
	}

	results, err := p.Search(context.Background(), "golang")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return 3 unique hits (4th is a duplicate of the 1st)
	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}

	if results[0].URL != "https://go.dev/doc/effective_go" || results[0].Title != "Effective Go" {
		t.Errorf("hit[0] = {%s, %s}, want {https://go.dev/doc/effective_go, Effective Go}", results[0].URL, results[0].Title)
	}
	if results[1].URL != "https://github.com/golang/go/wiki/CodeReviewComments" || results[1].Title != "Code Review Comments" {
		t.Errorf("hit[1] = {%s, %s}, want github URL", results[1].URL, results[1].Title)
	}
	if results[2].URL != "https://example.com/protocol-relative" || results[2].Title != "Protocol-relative link" {
		t.Errorf("hit[2] = {%s, %s}, want example.com URL", results[2].URL, results[2].Title)
	}
}

func TestSearchFallsBackToGenericLinks(t *testing.T) {
	// When no result__a hits exist, Search should fall back to generic <a> links.
	const noResultsHTML = `<!DOCTYPE html>
<html>
<head><title>No results</title></head>
<body>
<div class="no-results">Sorry, no results found for your search.</div>
<p>Try <a href="https://example.com/help">search help</a> or <a href="https://example.com/contact">contact us</a>.</p>
</body>
</html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(noResultsHTML))
	}))
	defer srv.Close()

	p := &Provider{
		client:    srv.Client(),
		searchURL: srv.URL,
	}

	results, err := p.Search(context.Background(), "noresults")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 fallback results, got %d", len(results))
	}
	if results[0].URL != "https://example.com/help" || results[0].Title != "search help" {
		t.Errorf("hit[0] = {%s, %s}", results[0].URL, results[0].Title)
	}
	if results[1].URL != "https://example.com/contact" || results[1].Title != "contact us" {
		t.Errorf("hit[1] = {%s, %s}", results[1].URL, results[1].Title)
	}
}

func TestMaxResultsTruncation(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><body>`)
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, `<a rel="nofollow" class="result__a" href="https://example.com/%d">Result %d</a>`, i, i)
	}
	b.WriteString(`</body></html>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(b.String()))
	}))
	defer srv.Close()

	p := &Provider{
		client:    srv.Client(),
		searchURL: srv.URL,
	}

	results, err := p.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) > maxResults {
		t.Errorf("got %d results, expected at most %d", len(results), maxResults)
	}
}

func TestDecodeDDGRedirect(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://example.com", "https://example.com"},
		{"//example.com/page", "https://example.com/page"},
		{"/l/?uddg=https%3A%2F%2Fgithub.com%2Fgo", "https://github.com/go"},
		{"/l/?uddg=https://en.wikipedia.org/wiki/Go_(programming_language)", "https://en.wikipedia.org/wiki/Go_(programming_language)"},
		{"ftp://not-http", ""},
	}

	for _, tt := range tests {
		got := decodeDDGRedirect(tt.raw)
		if got != tt.want {
			t.Errorf("decodeDDGRedirect(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestExtractQuotedValue(t *testing.T) {
	val, rest, ok := extractQuotedValue(`"hello" world`)
	if !ok {
		t.Fatal("expected to extract quoted value")
	}
	if val != "hello" {
		t.Errorf("val = %q, want %q", val, "hello")
	}
	if rest != " world" {
		t.Errorf("rest = %q, want %q", rest, " world")
	}
}
