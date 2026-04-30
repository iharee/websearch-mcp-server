package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iharee/websearch-mcp/internal/fetcher"
	"github.com/iharee/websearch-mcp/internal/searcher"
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
	Long: `Web search and content fetching for AI agents, without MCP protocol overhead.

Outputs LLM-friendly text to stdout. Exit code 0 on success, non-zero on error.

Proxy: Go's HTTP library reads proxy settings from HTTP_PROXY / HTTPS_PROXY
env vars only — it does not use the OS system proxy. For CDP browser proxy,
see README.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the web via DuckDuckGo or Tavily",
	Long: `Search the web and return results as LLM-friendly markdown.

Output is a list of [title](url) links. Prints a message if no results match.

Unknown --engine values are rejected with an error listing valid choices.`,
	Example: `  websearch-cli search "golang release notes"
  websearch-cli search "AI safety" --engine tavily
  SEARCH_ENGINE=tavily TAVILY_API_KEY=sk-... websearch-cli search "quantum computing"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := newContext(cmd)
		defer cancel()

		query := args[0]
		engine, _ := cmd.Flags().GetString("engine")

		p, err := searcher.Resolve(strings.ToLower(engine))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		results, err := p.Search(ctx, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "search failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(searcher.FormatResults(query, results))
	},
}

var fetchCmd = &cobra.Command{
	Use:   "fetch <url>",
	Short: "Fetch and extract page content by URL",
	Long: `Fetch a URL, convert HTML to readable text, and return clean content.

The --method flag selects the fetch backend:

  direct   Plain HTTP. Strips HTML tags. Fast, no external dependencies.
  cdp      Chrome DevTools Protocol. Renders JavaScript, extracts innerText.
           Use --cdp-mode to control how the browser is acquired:
             connect  Connect to an already-running Chrome (default).
             system   Find and launch your system Chrome/Chromium/Edge.
             bundled  Auto-download rod's Chromium on first use.

Unknown --method or --cdp-mode values are rejected with an error.

When behind a proxy, set HTTP_PROXY / HTTPS_PROXY for direct mode.
For cdp, Chrome does NOT read HTTP_PROXY — it uses the OS system proxy
(Windows Internet Options, macOS Network Preferences) or an explicit
--proxy-server flag (not set by default).`,
	Example: `  websearch-cli fetch https://example.com
  websearch-cli fetch https://example.com --method cdp --mode title
  websearch-cli fetch https://example.com --method cdp --cdp-mode bundled
  websearch-cli fetch https://example.com --method cdp --cdp-mode system`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := newContext(cmd)
		defer cancel()

		url := args[0]
		method, _ := cmd.Flags().GetString("method")

		cdpMode := ""
		if method == "cdp" || method == "" {
			cdpMode, _ = cmd.Flags().GetString("cdp-mode")
		}

		f, err := fetcher.Resolve(strings.ToLower(method), strings.ToLower(cdpMode))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		f.Warmup()

		mode, _ := cmd.Flags().GetString("mode")
		noCache, _ := cmd.Flags().GetBool("no-cache")

		result, err := f.Fetch(ctx, url, mode, noCache)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(fetcher.FormatResult(result))
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
	fetchCmd.Flags().Bool("no-cache", false, "Bypass the fetch cache and force a fresh request.")
	fetchCmd.Flags().String("cdp-mode", "", "CDP browser source when method=cdp: connect (default), system, or bundled. Defaults to CDP_MODE env var, or connect.")
}
