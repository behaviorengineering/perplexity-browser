// Command perplexity-browser-mcp is the stdio MCP server for Perplexity Pro browser research.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xynova/perplexity-browser/internal/config"
	"github.com/xynova/perplexity-browser/internal/mcptools"
	"github.com/xynova/perplexity-browser/internal/session"
)

const version = "0.1.0"

func main() {
	os.Exit(run())
}

func run() int {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	mgr := session.New(cfg, log)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "perplexity-browser",
		Version: version,
		Title:   "Perplexity Browser MCP",
	}, nil)
	mcptools.Register(server, mgr, log)

	log.Info("starting", "version", version, "user_data_dir", cfg.UserDataDir)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "perplexity-browser-mcp: %v\n", err)
		_ = mgr.CloseBrowser()
		return 1
	}
	_ = mgr.CloseBrowser()
	return 0
}
