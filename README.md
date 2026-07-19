# Perplexity Browser MCP

Go MCP server that drives **perplexity.ai** in a headed Playwright browser (Pro login, Deep research, continue, export).

**Status:** P2 — `perplexity_session`, `perplexity_research`, `perplexity_continue` wired; export in P3.

Consilium design: see that repo’s `docs/planned/perplexity-browser/`.

## Tools

| Tool | Status |
|------|--------|
| `perplexity_session` | **Yes** — `status`, `wait_for_login`, `close`, `cancel` |
| `perplexity_research` | **Yes** — new thread, mode (`deep`/`search`), submit, wait, extract |
| `perplexity_continue` | **Yes** — follow-up on active thread |
| `perplexity_export` | Stub until P3 |

## Setup

```bash
make bootstrap   # go mod tidy + Playwright Chromium (npm + nodejs driver assemble)
make build
make test
make smoke       # prints session JSON (need_login on a cold profile is OK)
```

Requires **playwright-go v0.6100.0+** (assembles driver from npm + nodejs.org). See [scripts/bootstrap-note.md](scripts/bootstrap-note.md).


Profile (dedicated Chromium user-data dir; **login cookies persist** across MCP restarts):

```text
~/.perplexity-browser-mcp/profile
~/.perplexity-browser-mcp/exports
```

If the browser window is closed, the next tool call relaunches from that same profile (you should not need to sign in again). `close` only stops the window; it does **not** wipe the profile.

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
