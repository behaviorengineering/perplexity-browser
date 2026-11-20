// Command smoke opens Perplexity headed and prints perplexity_session-style JSON.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/behaviorengineering/perplexity-browser/internal/config"
	"github.com/behaviorengineering/perplexity-browser/internal/session"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	mgr := session.New(cfg, log)
	defer func() { _ = mgr.CloseBrowser() }()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if len(args) > 0 && args[0] == "export" {
		return runExportSmoke(ctx, mgr)
	}
	if len(args) > 0 && args[0] == "dump" {
		return runDumpSmoke(ctx, mgr)
	}
	return runStatusSmoke(ctx, mgr)
}

func runStatusSmoke(ctx context.Context, mgr *session.Manager) int {
	s := mgr.Status(ctx, true, "")
	if err := encodeJSON(s); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		return 1
	}
	if s.Status == "error" {
		return 1
	}
	// need_login is a successful smoke (browser opened).
	return 0
}

func runExportSmoke(ctx context.Context, mgr *session.Manager) int {
	s := mgr.Status(ctx, true, "")
	if s.Status == "error" {
		_ = encodeJSON(s)
		return 1
	}
	if !s.LoggedIn {
		out := map[string]any{"status": s.Status, "reason": "need_login", "message": s.Message}
		_ = encodeJSON(out)
		return 0
	}
	probe, err := mgr.ProbeExtract(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe: %v\n", err)
		return 1
	}
	if err := encodeJSON(probe); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		return 1
	}
	if probe.Reason == "no_page" || probe.Reason == "no_conversation" || probe.Reason == "prompt_miss" {
		return 0
	}
	user := strings.TrimSpace(probe.UserAssembled)
	asst := strings.TrimSpace(probe.Assistant)
	if user != "" && asst != "" && strings.HasPrefix(asst, user) {
		fmt.Fprintf(os.Stderr, "smoke-export: assembled user is a prefix of assistant (issue 14 mislabel)\n")
		return 1
	}
	if asst == "" && strings.Contains(strings.ToLower(probe.URL), "/search/") {
		fmt.Fprintf(os.Stderr, "smoke-export: empty assistant on thread URL\n")
		return 1
	}
	return 0
}

func runDumpSmoke(ctx context.Context, mgr *session.Manager) int {
	s := mgr.Status(ctx, true, "")
	if s.Status == "error" {
		if err := encodeJSON(s); err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			return 1
		}
		return 1
	}
	if !s.LoggedIn {
		out := map[string]any{"status": s.Status, "reason": "need_login", "message": s.Message}
		if err := encodeJSON(out); err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			return 1
		}
		return 0
	}
	dump, err := mgr.ProbePromptDOM(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dump: %v\n", err)
		return 1
	}
	if err := encodeJSON(dump); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		return 1
	}
	return 0
}

func encodeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
