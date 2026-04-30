package direct

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iharee/websearch-mcp/internal/model"
)

const (
	userAgent           = "websearch-mcp/0.1"
	requestTimeout      = 20 * time.Second
	maxRedirects        = 10
	defaultPreviewChars = 900
	summaryPreviewChars = 1200
	titlePreviewChars   = 600
)

type Provider struct {
	client *http.Client
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
	}
}

func (p *Provider) Fetch(ctx context.Context, rawURL string, prompt string) (*model.FetchResult, error) {
	requestURL, err := normalizeURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	bodyStr := string(body)
	contentType := resp.Header.Get("Content-Type")

	text := selectContent(bodyStr, prompt, contentType)
	title := extractTitle(bodyStr, contentType)

	return &model.FetchResult{
		URL:     resp.Request.URL.String(),
		Title:   title,
		Content: text,
	}, nil
}

func normalizeURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	if parsed.Scheme == "http" {
		host := parsed.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			parsed.Scheme = "https"
		}
	}
	return parsed.String(), nil
}

func selectContent(body, prompt, contentType string) string {
	normalized := normalizeContent(body, contentType)
	compact := collapseWhitespace(normalized)
	lowerPrompt := strings.ToLower(prompt)

	if strings.Contains(lowerPrompt, "full") {
		return compact
	}
	if strings.Contains(lowerPrompt, "title") {
		return previewText(compact, titlePreviewChars)
	}
	if strings.Contains(lowerPrompt, "summary") || strings.Contains(lowerPrompt, "summarize") {
		return previewText(compact, summaryPreviewChars)
	}
	return previewText(compact, defaultPreviewChars)
}

func normalizeContent(body, contentType string) string {
	if strings.Contains(contentType, "html") {
		return htmlToText(body)
	}
	return strings.TrimSpace(body)
}

func extractTitle(body, contentType string) string {
	if strings.Contains(contentType, "html") {
		lower := strings.ToLower(body)
		start := strings.Index(lower, "<title>")
		if start == -1 {
			start = strings.Index(lower, "<title ")
		}
		if start != -1 {
			start += strings.Index(body[start:], ">") + 1
			end := strings.Index(lower, "</title>")
			if end != -1 && end > start {
				title := body[start:end]
				return collapseWhitespace(decodeEntities(strings.TrimSpace(title)))
			}
		}
	}
	text := normalizeContent(body, contentType)
	if line, _, found := strings.Cut(text, "\n"); found {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(text)
}

func htmlToText(html string) string {
	var buf strings.Builder
	buf.Grow(len(html))
	inTag := false

	for _, ch := range html {
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

	return decodeEntities(buf.String())
}

func decodeEntities(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func previewText(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return strings.TrimSpace(string(runes[:maxChars])) + "..."
}
