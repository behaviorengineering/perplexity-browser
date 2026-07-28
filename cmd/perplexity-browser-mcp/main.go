// Command perplexity-browser-mcp runs the stdio MCP server or repo init scaffolding.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/behaviorengineering/perplexity-browser/internal/bootstrap"
	"github.com/behaviorengineering/perplexity-browser/internal/config"
	"github.com/behaviorengineering/perplexity-browser/internal/mcptools"
	"github.com/behaviorengineering/perplexity-browser/internal/session"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) > 1 {
		switch args[1] {
		case "init":
			return runInit(args[2:])
		case "help", "-h", "--help":
			printUsage(os.Stdout)
			return 0
		case "version", "-v", "--version":
			fmt.Printf("perplexity-browser-mcp %s\n", version)
			return 0
		}
	}
	return runMCP()
}

func runMCP() int {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	mgr := session.New(cfg, log)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "perplexity-browser",
		Version: version,
		Title:   "Perplexity Browser MCP",
	}, nil)
	mcptools.Register(server, mgr, log)

	log.Info("starting", "version", version, "user_data_dir", cfg.UserDataDir, "session_id", cfg.SessionID, "sessions_dir", cfg.SessionsDir)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "perplexity-browser-mcp: %v\n", err)
		_ = mgr.CloseBrowser()
		return 1
	}
	_ = mgr.CloseBrowser()
	return 0
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		project  string
		workflow string
		layout   string
		pack     string
		mode     string
		force    bool
	)
	fs.StringVar(&project, "project", "", "project display name (default: repo directory name)")
	fs.StringVar(&workflow, "workflow", "", "workflow id (default: <repo>-perplexity-research)")
	fs.StringVar(&layout, "layout", bootstrap.LayoutCursor, "output layout: cursor (default) or docs")
	fs.StringVar(&pack, "pack", "deep-research", "initial pack filename slug")
	fs.StringVar(&mode, "mode", "deep", "Perplexity mode for the pack: deep or search")
	fs.BoolVar(&force, "force", false, "overwrite existing workflow and pack files")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repo := "."
	if fs.NArg() > 0 {
		repo = fs.Arg(0)
	}

	res, err := bootstrap.Init(bootstrap.Options{
		RepoRoot:    repo,
		ProjectName: project,
		WorkflowID:  workflow,
		Layout:      layout,
		PackSlug:    pack,
		Mode:        mode,
		Force:       force,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "perplexity-browser-mcp init: %v\n", err)
		return 1
	}
	bootstrap.WriteHandoff(os.Stdout, res)
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `perplexity-browser-mcp %s

Usage:
  perplexity-browser-mcp              Start MCP stdio server (default)
  perplexity-browser-mcp init [dir]   Scaffold Perplexity workflow files in a repo
  perplexity-browser-mcp help         Show this help
  perplexity-browser-mcp version      Print version

init flags:
  --project string    Project display name (default: directory name)
  --workflow string   Workflow id (default: <dir>-perplexity-research)
  --layout string     cursor (default) or docs
  --pack string       Pack slug (default: deep-research)
  --mode string       deep (default) or search
  --force             Overwrite existing files

Examples:
  perplexity-browser-mcp init
  perplexity-browser-mcp init /path/to/my-repo --project "My App"
  perplexity-browser-mcp init . --layout docs

`, version)
}
