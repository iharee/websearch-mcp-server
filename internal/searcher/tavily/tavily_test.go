package tavily

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}

		var req tavilySearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Query != "Why is Saki so moe?" {
			t.Errorf("unexpected query: %s", req.Query)
		}

		json.NewEncoder(w).Encode(tavilySearchResponse{
			Results: []tavilyResult{
				{Title: "Saki - Wikipedia", URL: "https://example.com/saki", Content: "Saki..."},
				{Title: "Why is Saki so moe?", URL: "https://example.com/saki-moe", Content: "Saki is cute..."},
			},
		})
	}))
	defer srv.Close()

	p := &Provider{
		client:      srv.Client(),
		apiKey:      "test-key",
		searchDepth: "basic",
		maxResults:  7,
		topic:       "general",
		baseURL:     srv.URL,
	}

	results, err := p.Search(context.Background(), "Why is Saki so moe?")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}

	if results[0].URL != "https://en.wikipedia.org/wiki/Saki" {
		t.Errorf("result[0].URL = %q, want wikipedia URL", results[0].URL)
	}
	if results[0].Title != "Saki - Wikipedia" {
		t.Errorf("result[0].Title = %q", results[0].Title)
	}
	if results[0].Snippet != "Saki is a..." {
		t.Errorf("result[0].Snippet = %q", results[0].Snippet)
	}

	if results[1].URL != "https://example.com/saki-moe" {
		t.Errorf("result[1].URL = %q", results[1].URL)
	}
	if results[1].Title != "Why is Saki so popular?" {
		t.Errorf("result[1].Title = %q", results[1].Title)
	}
	if results[1].Snippet != "Saki is popular because..." {
		t.Errorf("result[1].Snippet = %q", results[1].Snippet)
	}
}

func TestSearchEmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tavilySearchResponse{Results: []tavilyResult{}})
	}))
	defer srv.Close()

	p := &Provider{
		client:      srv.Client(),
		apiKey:      "test-key",
		searchDepth: "basic",
		maxResults:  7,
		topic:       "general",
		baseURL:     srv.URL,
	}

	results, err := p.Search(context.Background(), "noresults")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestSearchAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid API key"}`))
	}))
	defer srv.Close()

	p := &Provider{
		client:      srv.Client(),
		apiKey:      "bad-key",
		searchDepth: "basic",
		maxResults:  7,
		topic:       "general",
		baseURL:     srv.URL,
	}

	_, err := p.Search(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if err.Error() != "tavily authentication failed: check TAVILY_API_KEY" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestSearchRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	p := &Provider{
		client:      srv.Client(),
		apiKey:      "test-key",
		searchDepth: "basic",
		maxResults:  7,
		topic:       "general",
		baseURL:     srv.URL,
	}

	_, err := p.Search(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
	if err.Error() != "tavily rate limit exceeded, retry later" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestSearchMissingAPIKey(t *testing.T) {
	p := &Provider{
		client:      &http.Client{},
		apiKey:      "",
		searchDepth: "basic",
		maxResults:  7,
		topic:       "general",
		baseURL:     "https://api.tavily.com",
	}

	_, err := p.Search(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
	if err.Error() != "tavily authentication failed: TAVILY_API_KEY is not set" {
		t.Errorf("unexpected error: %s", err)
	}
}
