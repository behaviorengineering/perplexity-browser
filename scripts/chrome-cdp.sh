#!/usr/bin/env bash
# Launch Google Chrome with remote debugging so the MCP can ConnectOverCDP.
# 1) Quit any Chrome using this profile (including the Playwright-launched one).
# 2) Run this script.
# 3) Complete Cloudflare / login in the window until you see normal Perplexity UI.
# 4) Reload MCP with PERPLEXITY_BROWSER_CDP_URL=http://127.0.0.1:9222 and call status.

set -euo pipefail

PROFILE="${PERPLEXITY_BROWSER_USER_DATA_DIR:-$HOME/.perplexity-browser-mcp/profile-chrome}"
PORT="${PERPLEXITY_BROWSER_CDP_PORT:-9222}"
CHROME="${PERPLEXITY_BROWSER_CHROME_APP:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
URL="${PERPLEXITY_BROWSER_BASE_URL:-https://www.perplexity.ai}"

mkdir -p "$PROFILE"
echo "CDP port: $PORT"
echo "Profile:  $PROFILE"
echo "Once Perplexity looks normal (compose box visible), keep this Chrome open and use the MCP."

exec "$CHROME" \
  --remote-debugging-port="$PORT" \
  --user-data-dir="$PROFILE" \
  --no-first-run \
  --no-default-browser-check \
  --disable-features=Translate \
  "$URL"
