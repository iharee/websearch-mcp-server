package duckduckgo

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iharee/websearch-mcp-server/internal/model"
)

const (
	defaultSearchURL = "https://html.duckduckgo.com/html/"
	userAgent        = "websearch-mcp-server/0.1"
	requestTimeout   = 20 * time.Second
	maxRedirects     = 10
	maxResults       = 8
)

type Provider struct {
	client    *http.Client
	searchURL string
}

func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: requestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("stopped after %d redirects", maxRedirects)
				}
				return nil
			},
		},
		searchURL: defaultSearchURL,
	}
}

func (p *Provider) Search(ctx context.Context, query string) ([]model.SearchResult, error) {
	u, err := url.Parse(p.searchURL)
	if err != nil {
		return nil, fmt.Errorf("parse search URL: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	htmlStr := string(body)
	hits := extractSearchHits(htmlStr)

	if len(hits) == 0 {
		hits = extractGenericLinks(htmlStr)
	}

	dedupeHits(&hits)
	if len(hits) > maxResults {
		hits = hits[:maxResults]
	}

	return hits, nil
}

func extractSearchHits(htmlStr string) []model.SearchResult {
	var hits []model.SearchResult
	remaining := htmlStr

	for {
		anchorStart := strings.Index(remaining, "result__a")
		if anchorStart == -1 {
			break
		}
		afterClass := remaining[anchorStart:]

		hrefIdx := strings.Index(afterClass, "href=")
		if hrefIdx == -1 {
			remaining = afterClass[1:]
			continue
		}
		hrefSlice := afterClass[hrefIdx+5:]
		hrefURL, rest, ok := extractQuotedValue(hrefSlice)
		if !ok {
			remaining = afterClass[1:]
			continue
		}

		closeTagIdx := strings.IndexByte(rest, '>')
		if closeTagIdx == -1 {
			remaining = afterClass[1:]
			continue
		}
		afterTag := rest[closeTagIdx+1:]

		endAnchorIdx := strings.Index(afterTag, "</a>")
		if endAnchorIdx == -1 {
			remaining = afterTag[1:]
			continue
		}

		title := htmlToText(afterTag[:endAnchorIdx])
		if decodedURL := decodeDDGRedirect(hrefURL); decodedURL != "" && title != "" {
			hits = append(hits, model.SearchResult{
				URL:   decodedURL,
				Title: strings.TrimSpace(title),
			})
		}
		remaining = afterTag[endAnchorIdx+4:]
	}
	return hits
}

func extractGenericLinks(htmlStr string) []model.SearchResult {
	var hits []model.SearchResult
	remaining := htmlStr

	for {
		anchorStart := strings.Index(remaining, "<a")
		if anchorStart == -1 {
			break
		}
		afterAnchor := remaining[anchorStart:]

		hrefIdx := strings.Index(afterAnchor, "href=")
		if hrefIdx == -1 {
			remaining = afterAnchor[2:]
			continue
		}
		hrefSlice := afterAnchor[hrefIdx+5:]
		hrefURL, rest, ok := extractQuotedValue(hrefSlice)
		if !ok {
			remaining = afterAnchor[2:]
			continue
		}

		closeTagIdx := strings.IndexByte(rest, '>')
		if closeTagIdx == -1 {
			remaining = afterAnchor[2:]
			continue
		}
		afterTag := rest[closeTagIdx+1:]

		endAnchorIdx := strings.Index(afterTag, "</a>")
		if endAnchorIdx == -1 {
			remaining = afterAnchor[2:]
			continue
		}

		title := strings.TrimSpace(htmlToText(afterTag[:endAnchorIdx]))
		if title == "" {
			remaining = afterTag[endAnchorIdx+4:]
			continue
		}

		decodedURL := decodeDDGRedirect(hrefURL)
		if decodedURL == "" {
			decodedURL = hrefURL
		}
		if strings.HasPrefix(decodedURL, "http://") || strings.HasPrefix(decodedURL, "https://") {
			hits = append(hits, model.SearchResult{
				URL:   decodedURL,
				Title: title,
			})
		}
		remaining = afterTag[endAnchorIdx+4:]
	}
	return hits
}

func extractQuotedValue(s string) (value string, rest string, ok bool) {
	if len(s) == 0 {
		return "", "", false
	}
	quote := s[0]
	if quote != '"' && quote != '\'' {
		return "", "", false
	}
	rest = s[1:]
	end := strings.IndexByte(rest, quote)
	if end == -1 {
		return "", "", false
	}
	value = rest[:end]
	rest = rest[end+1:]
	return value, rest, true
}

func decodeDDGRedirect(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return html.UnescapeString(rawURL)
	}

	var joined string
	if strings.HasPrefix(rawURL, "//") {
		joined = "https:" + rawURL
	} else if strings.HasPrefix(rawURL, "/") {
		joined = "https://duckduckgo.com" + rawURL
	} else {
		return ""
	}

	parsed, err := url.Parse(joined)
	if err != nil {
		return ""
	}

	if parsed.Path == "/l/" || parsed.Path == "/l" {
		if uddg := parsed.Query().Get("uddg"); uddg != "" {
			return html.UnescapeString(uddg)
		}
	}
	return html.UnescapeString(joined)
}

func htmlToText(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	inTag := false

	for _, ch := range s {
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
		case inTag:
		default:
			buf.WriteRune(ch)
		}
	}

	parts := strings.Fields(html.UnescapeString(buf.String()))
	return strings.Join(parts, " ")
}

func dedupeHits(hits *[]model.SearchResult) {
	seen := make(map[string]bool)
	n := 0
	for _, hit := range *hits {
		if !seen[hit.URL] {
			seen[hit.URL] = true
			(*hits)[n] = hit
			n++
		}
	}
	*hits = (*hits)[:n]
}
