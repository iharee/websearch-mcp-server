package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/iharee/websearch-mcp/internal/config"
	"github.com/iharee/websearch-mcp/internal/fetcher"
	"github.com/iharee/websearch-mcp/internal/mcp"
	"github.com/iharee/websearch-mcp/internal/searcher"
)

func main() {
	var portFlag string
	flag.StringVar(&portFlag, "port", "", "Server listen port (overrides PORT env var, default 8848)")
	flag.StringVar(&portFlag, "p", "", "Short form of --port")
	flag.Parse()

	cfg := config.Load(portFlag)

	srv := mcp.New()

	srv.RegisterTool(searcher.ToolDefinition(), searcher.Handler())
	srv.RegisterTool(fetcher.ToolDefinition(), fetcher.Handler())

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/mcp", srv.ServeHTTP)

	addr := fmt.Sprintf(":%s", cfg.Port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("MCP server starting on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	log.Println("server shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	fetcher.Shutdown()

	log.Println("server stopped")
}
