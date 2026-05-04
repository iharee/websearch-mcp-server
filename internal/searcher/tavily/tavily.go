package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/iharee/websearch-mcp/internal/model"
)

const (
	defaultBaseURL     = "https://api.tavily.com"
	defaultSearchPath  = "/search"
	defaultSearchDepth = "basic"
	defaultMaxResults  = 7
	defaultTopic       = "general"
	requestTimeout     = 20 * time.Second
)

type Provider struct {
	client      *http.Client
	apiKey      string
	searchDepth string
	maxResults  int
	topic       string
	baseURL     string
}

func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		apiKey:      os.Getenv("TAVILY_API_KEY"),
		searchDepth: envOrDefault("TAVILY_SEARCH_DEPTH", defaultSearchDepth),
		maxResults:  envIntOrDefault("TAVILY_MAX_RESULTS", defaultMaxResults),
		topic:       envOrDefault("TAVILY_TOPIC", defaultTopic),
		baseURL:     defaultBaseURL,
	}
}

func (p *Provider) Search(ctx context.Context, query string) ([]model.SearchResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("tavily authentication failed: TAVILY_API_KEY is not set")
	}

	reqBody := tavilySearchRequest{
		Query:       query,
		SearchDepth: p.searchDepth,
		MaxResults:  p.maxResults,
		Topic:       p.topic,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("tavily marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+defaultSearchPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("tavily build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, tavilyError(resp)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tavily read response: %w", err)
	}

	var searchResp tavilySearchResponse
	if err := json.Unmarshal(respBytes, &searchResp); err != nil {
		return nil, fmt.Errorf("tavily parse response: %w", err)
	}

	results := make([]model.SearchResult, 0, len(searchResp.Results))
	for _, r := range searchResp.Results {
		results = append(results, model.SearchResult{
			URL:     r.URL,
			Title:   r.Title,
			Snippet: r.Content,
		})
	}

	return results, nil
}

type tavilySearchRequest struct {
	Query       string `json:"query"`
	SearchDepth string `json:"search_depth,omitempty"`
	MaxResults  int    `json:"max_results,omitempty"`
	Topic       string `json:"topic,omitempty"`
}

type tavilySearchResponse struct {
	Results      []tavilyResult `json:"results"`
	ResponseTime float64        `json:"response_time"`
}

type tavilyResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

func tavilyError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	detail := strings.TrimSpace(string(body))
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("tavily bad request: %s", detail)
	case http.StatusUnauthorized:
		return fmt.Errorf("tavily authentication failed: check TAVILY_API_KEY")
	case http.StatusTooManyRequests:
		return fmt.Errorf("tavily rate limit exceeded, retry later")
	case http.StatusInternalServerError:
		return fmt.Errorf("tavily server error: %s", detail)
	default:
		return fmt.Errorf("tavily unexpected status %d: %s", resp.StatusCode, detail)
	}
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return defaultValue
}

func envIntOrDefault(key string, defaultValue int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultValue
	}
	return n
}
