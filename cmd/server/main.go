package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/iharee/websearch-mcp-server/internal/config"
	"github.com/iharee/websearch-mcp-server/internal/fetcher"
	"github.com/iharee/websearch-mcp-server/internal/mcp"
	"github.com/iharee/websearch-mcp-server/internal/searcher"
)

func main() {
	cfg := config.Load()

	srv := mcp.New()

	srv.RegisterTool(searcher.ToolDefinition(), searcher.Handler())

	contentFetcher := &fetcher.MockFetcher{}
	srv.RegisterTool(fetcher.ToolDefinition(), fetcher.Handler(contentFetcher))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/mcp", srv.ServeHTTP)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("MCP server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
