# Perplexity Browser MCP

Go MCP server that drives **perplexity.ai** in a headed Playwright browser (Pro login, Deep research, continue, export).

**Status:** P1 skeleton — `perplexity_session` works; `research` / `continue` / `export` registered, automation in P2/P3.

Consilium design: see that repo’s `docs/planned/perplexity-browser/`.

## Tools

| Tool | P1 |
|------|----|
| `perplexity_session` | **Yes** — `status`, `wait_for_login`, `close`, `cancel` |
| `perplexity_research` | Opens session; returns `not_ready` until P2 |
| `perplexity_continue` | Stub until P2 |
| `perplexity_export` | Stub until P3 |

## Setup

```bash
make bootstrap   # go mod tidy + Playwright Chromium driver
make build
make test
make smoke       # headed browser → prints session JSON (need_login is OK)
```

If `make bootstrap` fails downloading the Go Playwright **driver** zip, see [scripts/bootstrap-note.md](scripts/bootstrap-note.md). Unit tests and `make build` still work.


Profile (dedicated; not your daily Chrome profile):

```text
~/.perplexity-browser-mcp/profile
~/.perplexity-browser-mcp/exports
```

## Cursor MCP config

```json
{
  "mcpServers": {
    "perplexity-browser": {
      "command": "/absolute/path/to/perplexity-browser/bin/perplexity-browser-mcp",
      "env": {
        "PERPLEXITY_BROWSER_HEADLESS": "0"
      }
    }
  }
}
```

From a Consilium checkout after submodule + build:

```text
providers/perplexity-browser/bin/perplexity-browser-mcp
```

## Env

| Variable | Default |
|----------|---------|
| `PERPLEXITY_BROWSER_USER_DATA_DIR` | `~/.perplexity-browser-mcp/profile` |
| `PERPLEXITY_BROWSER_EXPORT_DIR` | `~/.perplexity-browser-mcp/exports` |
| `PERPLEXITY_BROWSER_HEADLESS` | `0` |
| `PERPLEXITY_BROWSER_BASE_URL` | `https://www.perplexity.ai` |
| `PERPLEXITY_BROWSER_DEFAULT_TIMEOUT_MS` | `900000` |

## License

Apache-2.0
