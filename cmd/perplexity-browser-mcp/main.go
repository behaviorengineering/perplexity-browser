// Command perplexity-browser-mcp runs the stdio MCP server or repo init scaffolding.
package main

import (
	"os"

	"github.com/behaviorengineering/perplexity-browser/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args))
}
