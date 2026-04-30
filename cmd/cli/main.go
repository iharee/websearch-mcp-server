package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/fetcher"
	"github.com/iharee/websearch-mcp/internal/fetcher/cdp"
	"github.com/iharee/websearch-mcp/internal/fetcher/direct"
	"github.com/iharee/websearch-mcp/internal/searcher"
	"github.com/iharee/websearch-mcp/internal/searcher/duckduckgo"
	"github.com/iharee/websearch-mcp/internal/searcher/tavily"
)

const defaultTimeout = 30 * time.Second

func main() {
	rootCmd.AddCommand(searchCmd, fetchCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "websearch-cli",
	Short: "Advanced web search tools for agents via CLI",
	Long: `A CLI for web search and content fetching, designed for direct invocation
by AI agents without MCP protocol overhead.

Environment Variables:
  SEARCH_ENGINE    Search engine (default duckduckgo): duckduckgo or tavily
  FETCH_METHOD     Fetch method (default direct): direct or cdp
  TAVILY_API_KEY   API key for Tavily search

Priority: explicit flag > environment variable > built-in default.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the web via DuckDuckGo or Tavily",
	Long: `Search the web using the specified engine and return results as
LLM-friendly markdown.

Arguments:
  <query>    Search query string (required).

Output is a markdown list with [title](url) links. If no results
match, prints a message to stdout.

Failure Cases:
  The SEARCH_ENGINE env var or --engine flag holds an unknown engine
  value: falls back to duckduckgo.
  Network error or timeout: prints the error to stderr and exits with
  code 1.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := newContext(cmd)
		defer cancel()

		query := args[0]
		engine, _ := cmd.Flags().GetString("engine")
		if engine == "" {
			engine = config.SearchEngine()
		}

		var p searcher.Provider
		switch engine {
		case "tavily":
			p = tavily.NewProvider()
		default:
			p = duckduckgo.NewProvider()
		}

		results, err := p.Search(ctx, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "search failed: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Printf("No web search results matched the query %q.\n", query)
			return
		}

		fmt.Printf("Search results for %q. Include a Sources section in the final answer.\n", query)
		for _, r := range results {
			fmt.Printf("- [%s](%s)\n", r.Title, r.URL)
		}
	},
}

var fetchCmd = &cobra.Command{
	Use:   "fetch <url>",
	Short: "Fetch and extract page content by URL",
	Long: `Fetch a URL, convert HTML to readable text, and return content as
LLM-friendly text.

Arguments:
  <url>    Page URL to fetch (required).

Options:
  --method, -m    Fetch method: direct (plain HTTP) or cdp (Chrome DevTools
                  with JS rendering). Defaults to FETCH_METHOD env var.
  --mode, -o      Content length mode. One of:
                    full    — complete page content (untruncated)
                    summary — longer preview (~1200 chars)
                    title   — short preview (~600 chars)
                  Defaults to a 900-char preview if unset.

Output is the page title, URL, and plain-text content.

Failure Cases:
  The FETCH_METHOD env var or --method flag holds an unknown value:
  falls back to direct.
  Network error or timeout: prints the error to stderr and exits with
  code 1.
  The URL is missing a scheme (e.g. example.com): https:// is
  prepended automatically.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := newContext(cmd)
		defer cancel()

		url := args[0]
		method, _ := cmd.Flags().GetString("method")
		if method == "" {
			method = config.FetchMethod()
		}

		var p fetcher.Provider
		switch method {
		case "cdp":
			p = cdp.NewProvider()
		default:
			p = direct.NewProvider()
		}

		mode, _ := cmd.Flags().GetString("mode")

		result, err := p.Fetch(ctx, url, mode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Title: %s\n", result.Title)
		fmt.Printf("URL: %s\n", result.URL)
		if result.Content != "" {
			fmt.Printf("\n%s", result.Content)
		}
	},
}

func newContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	sigCtx, sigCancel := signal.NotifyContext(ctx, os.Interrupt)
	return sigCtx, func() {
		sigCancel()
		cancel()
	}
}

func init() {
	rootCmd.PersistentFlags().DurationP("timeout", "t", defaultTimeout,
		fmt.Sprintf("Request timeout (default %s)", defaultTimeout),
	)

	searchCmd.Flags().StringP("engine", "e", "", "Search engine (duckduckgo or tavily). Defaults to SEARCH_ENGINE env var, or duckduckgo.")
	fetchCmd.Flags().StringP("method", "m", "", "Fetch method: direct (plain HTTP, strips HTML) or cdp (Chrome DevTools, renders JS). Defaults to FETCH_METHOD env var, or direct.")
	fetchCmd.Flags().StringP("mode", "o", "", "Content length mode: full (complete), summary (longer preview), title (short preview). Defaults to a 900-char preview if unset.")
}
