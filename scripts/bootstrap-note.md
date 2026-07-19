# Bootstrap note

## Fixed (P1.1)

Upgrade to **`github.com/mxschmitt/playwright-go@v0.6100.0`**. That release **assembles** the driver from:

- `playwright-core` on the npm registry
- Node.js from `nodejs.org`

It does **not** use the retired `/builds/driver` CDN (`playwright.azureedge.net`), which returns 404/400.

```bash
make bootstrap   # go mod tidy + playwright install chromium
make smoke       # headed by default; HEADLESS=1 for CI-ish
```

`need_login` from smoke is success for a cold profile: browser opened and hit perplexity.ai.

## Env (optional)

| Variable | Purpose |
|----------|---------|
| `PLAYWRIGHT_NODEJS_PATH` | Use a preinstalled Node; skip Node download |
| `PLAYWRIGHT_GO_NPM_REGISTRY` | npm registry mirror for playwright-core |
| `NODE_MIRROR` | Node.js dist mirror |
| `PERPLEXITY_BROWSER_HEADLESS` | `1` for headless smoke |

## History

Before v0.6100.0, Go bindings downloaded a platform zip from azureedge mirrors. Microsoft stopped publishing those zips; installs failed with 404.
