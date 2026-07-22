# Perplexity Browser MCP

Go MCP server that drives **perplexity.ai** in a headed Playwright browser (Pro login, Deep research, continue, export).

**Status:** P3 — session, research, continue, and export are wired.

Consilium design: see that repo’s `docs/planned/perplexity-browser/`.

## Tools

| Tool | Status |
|------|--------|
| `perplexity_session` | **Yes** — `status`, `wait_for_login`, `close`, `cancel` |
| `perplexity_research` | **Yes** — new thread, mode (`deep`/`search`), submit, wait, extract |
| `perplexity_continue` | **Yes** — follow-up on active thread |
| `perplexity_export` | **Yes** — markdown file under export dir (`ui_export` or `scrape`) |

## Setup

```bash
make bootstrap   # go mod tidy + Playwright Chromium (npm + nodejs driver assemble)
make build
make test
make smoke       # prints session JSON (need_login on a cold profile is OK)
```

Requires **playwright-go v0.6100.0+** (assembles driver from npm + nodejs.org). See [scripts/bootstrap-note.md](scripts/bootstrap-note.md).


Profile (dedicated Chromium/Chrome user-data dir; **login cookies persist** across MCP restarts):

```text
~/.perplexity-browser-mcp/profile          # default (bundled Chromium)
~/.perplexity-browser-mcp/profile-chrome   # recommended with PERPLEXITY_BROWSER_CHANNEL=chrome
~/.perplexity-browser-mcp/exports
```

If Cloudflare / “security verification” blocks you on **Chromium for Testing**, use real Chrome over CDP:

```json
"env": {
  "PERPLEXITY_BROWSER_HEADLESS": "0",
  "PERPLEXITY_BROWSER_USER_DATA_DIR": "/Users/YOU/.perplexity-browser-mcp/profile-chrome",
  "PERPLEXITY_BROWSER_CDP_URL": "http://127.0.0.1:9222"
}
```

When `PERPLEXITY_BROWSER_CDP_URL` is set, the MCP **auto-launches Google Chrome** with the same flags as `scripts/chrome-cdp.sh` if nothing is listening on that port (`PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH=1`, default). Playwright-go then attaches over CDP. `close` disconnects only; Chrome stays open for login. Set `PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH=0` to require a manual `chrome-cdp.sh` start.

Alternative without CDP (Playwright launches Chromium/Chrome directly):

```json
"env": {
  "PERPLEXITY_BROWSER_HEADLESS": "0",
  "PERPLEXITY_BROWSER_CHANNEL": "chrome",
  "PERPLEXITY_BROWSER_USER_DATA_DIR": "/Users/YOU/.perplexity-browser-mcp/profile-chrome"
}
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
| `PERPLEXITY_BROWSER_CDP_URL` | (empty) — when set, attach over CDP instead of Playwright launch |
| `PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH` | `1` — launch Chrome like `scripts/chrome-cdp.sh` when CDP connect fails |
| `PERPLEXITY_BROWSER_CHROME_APP` | OS default Google Chrome path |

## License

Apache-2.0
