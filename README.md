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
| `perplexity_export` | **Yes** — Share → markdown into export dir (`ui_export` only; `export_manual` if UI fails) |

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

When `PERPLEXITY_BROWSER_CDP_URL` is set, the MCP **auto-launches Google Chrome** with the same flags as `scripts/chrome-cdp.sh` if nothing is listening on that port (`PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH=1`, default). Playwright-go then attaches over CDP. If the restored tab is blank or off-site (Chrome often ignores the startup URL with a reused profile), `status` / `wait_for_login` navigate once to Perplexity; an already-open Perplexity tab is left alone. `close` disconnects only; Chrome stays open for login. Set `PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH=0` to require a manual `chrome-cdp.sh` start.

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

## Multi-session (global MCP entry)

One **global** MCP entry is fine. The server shares one browser profile but **isolates thread state per Cursor workspace**:

1. **Auto (default):** MCP client `roots/list` → workspace folder name (e.g. `cr-case-intake` → `sessions/cr-case-intake.json`).
2. **Override:** pass `session_id` on a tool call.
3. **Fallback:** `PERPLEXITY_BROWSER_SESSION_ID` env when roots are unavailable (smoke CLI, non-Cursor clients).

`continue` / `export` activate the scope (navigate to that session URL if needed). Responses include `session_id` and `active_session_id`.

If two repos share the same folder basename, pass an explicit `session_id` on one of them.

## Repo integration (workflow layer)

The MCP is **repo-agnostic**. Each consumer project needs a **workflow doc** (prepare → research → export → file results).

**Scaffold in a consumer repo:**

```bash
perplexity-browser-mcp init /path/to/repo
perplexity-browser-mcp init . --layout docs
```

See **[docs/repo-integration.md](docs/repo-integration.md)** for layouts, agent customization, and smoke test.

| Command | Purpose |
|---------|---------|
| `perplexity-browser-mcp init [dir]` | Write workflow + pack templates (default: `.cursor/skills/<folder>-perplexity-research/`) |
| `perplexity-browser-mcp help` | Init flags and examples |

Reference consumer (domain-specific): Consilium `.cursor/skills/perplexity-browser-research/`.

## Env

| Variable | Default |
|----------|---------|
| `PERPLEXITY_BROWSER_USER_DATA_DIR` | `~/.perplexity-browser-mcp/profile` |
| `PERPLEXITY_BROWSER_EXPORT_DIR` | `~/.perplexity-browser-mcp/exports` |
| `PERPLEXITY_BROWSER_STATE_PATH` | `~/.perplexity-browser-mcp/state.json` (legacy; migrated for `default` session) |
| `PERPLEXITY_BROWSER_SESSION_ID` | `default` — default logical session when tools omit `session_id` |
| `PERPLEXITY_BROWSER_SESSIONS_DIR` | `~/.perplexity-browser-mcp/sessions` — one `\<session_id\>.json` per caller |
| `PERPLEXITY_BROWSER_HEADLESS` | `0` |
| `PERPLEXITY_BROWSER_BASE_URL` | `https://www.perplexity.ai` |
| `PERPLEXITY_BROWSER_DEFAULT_TIMEOUT_MS` | `900000` |
| `PERPLEXITY_BROWSER_SEARCH_TIMEOUT_MS` | `180000` |
| `PERPLEXITY_BROWSER_POLL_MS` | `800` (idle/stability poll) |
| `PERPLEXITY_BROWSER_POLL_FAST_MS` | `350` (poll while generating) |
| `PERPLEXITY_BROWSER_STABLE_POLLS` | `2` |
| `PERPLEXITY_BROWSER_CDP_URL` | (empty) — when set, attach over CDP instead of Playwright launch |
| `PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH` | `1` — launch Chrome like `scripts/chrome-cdp.sh` when CDP connect fails |
| `PERPLEXITY_BROWSER_CHROME_APP` | OS default Google Chrome path |

## License

Apache-2.0
