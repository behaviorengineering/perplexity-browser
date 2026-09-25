---
name: perplexity-browser-ops
description: >-
  Build and smoke the Perplexity Browser MCP server (Playwright). Use when
  installing Chromium, building bin/perplexity-browser-mcp, or debugging MCP
  session/smoke failures.
---

# Perplexity Browser MCP ops

**Module:** `github.com/behaviorengineering/perplexity-browser`

## Commands

MCP clients MUST invoke `perplexity-browser-mcp serve` (bare binary prints agent guide only).

```bash
make bootstrap   # go mod tidy + Playwright Chromium
make build       # bin/perplexity-browser-mcp
make test
make smoke       # session status smoke
make install     # copy binary to GOBIN
make ci          # tidy + gofmt + vet + race + build
```

Research workflow (MCP tools): load `perplexity-browser-research`. Host products MAY keep `.cursor/perplexity/` overlays for persona/packs outside this skill tree.
