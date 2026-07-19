// Command smoke opens Perplexity headed and prints perplexity_session-style JSON.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/xynova/perplexity-browser/internal/config"
	"github.com/xynova/perplexity-browser/internal/session"
)

func main() {
	os.Exit(run())
}

func run() int {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	mgr := session.New(cfg, log)
	defer func() { _ = mgr.CloseBrowser() }()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	s := mgr.Status(ctx, true)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		return 1
	}
	if s.Status == "error" {
		return 1
	}
	// need_login is a successful smoke (browser opened).
	return 0
}
