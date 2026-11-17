#!/bin/sh
# Start perplexity-browser MCP stdio server (plugin and manual installs).
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if command -v perplexity-browser-mcp >/dev/null 2>&1; then
  exec perplexity-browser-mcp serve "$@"
fi

if [ -x "$REPO_ROOT/bin/perplexity-browser-mcp" ]; then
  exec "$REPO_ROOT/bin/perplexity-browser-mcp" serve "$@"
fi

if command -v go >/dev/null 2>&1; then
  exec go run -C "$REPO_ROOT" ./cmd/perplexity-browser-mcp serve "$@"
fi

echo "perplexity-browser plugin: no binary. From the plugin clone run: make build   or   make install" >&2
exit 1