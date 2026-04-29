package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/iharee/websearch-mcp-server/internal/config"
	"github.com/iharee/websearch-mcp-server/internal/searcher"
	"github.com/iharee/websearch-mcp-server/internal/searcher/duckduckgo"
	"github.com/iharee/websearch-mcp-server/internal/searcher/tavily"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: websearch-cli <command> [args]")
		fmt.Fprintln(os.Stderr, "  search  --query <q> [--engine duckduckgo|tavily]")
		fmt.Fprintln(os.Stderr, "  fetch   --url <url>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "search":
		runSearch(os.Args[2:])
	case "fetch":
		fmt.Fprintln(os.Stderr, "fetch: not yet implemented")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runSearch(args []string) {
	query := flagArg(args, "query")
	engine := flagArg(args, "engine")
	if query == "" {
		fmt.Fprintln(os.Stderr, "search: --query is required")
		os.Exit(1)
	}
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

	results, err := p.Search(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search failed: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(results)
}

func flagArg(args []string, name string) string {
	prefix := "--" + name + "="
	for _, a := range args {
		if len(a) >= len(prefix) && a[:len(prefix)] == prefix {
			return a[len(prefix):]
		}
	}
	prefix = "--" + name
	for i, a := range args {
		if a == prefix && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
