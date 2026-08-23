#!/usr/bin/env bash
# Launch Google Chrome with remote debugging so the MCP can ConnectOverCDP.
# Optional: the MCP auto-launches Chrome with the same flags when CDP connect fails
# (PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH=1, default). Use this script to start Chrome
# manually or when another process already holds the profile.
# 1) Quit any Chrome using this profile (including the Playwright-launched one).
# 2) Run this script.
# 3) Complete Cloudflare / login in the window until you see normal Perplexity UI.
# 4) Reload MCP with PERPLEXITY_BROWSER_CDP_URL=http://127.0.0.1:9222 and call status.
#
# Cold start opens WarmupURL (default google.com); the MCP then navigates to Perplexity.

set -euo pipefail

PROFILE="${PERPLEXITY_BROWSER_USER_DATA_DIR:-$HOME/.perplexity-browser-mcp/profile-chrome}"
PORT="${PERPLEXITY_BROWSER_CDP_PORT:-9222}"
CHROME="${PERPLEXITY_BROWSER_CHROME_APP:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
WARMUP="${PERPLEXITY_BROWSER_WARMUP_URL:-https://www.google.com}"
BASE="${PERPLEXITY_BROWSER_BASE_URL:-https://www.perplexity.ai}"

# off/none/0/false disables warmup and opens Perplexity directly.
case "$(printf '%s' "$WARMUP" | tr '[:upper:]' '[:lower:]')" in
  off|none|0|false|no|disable|disabled) START_URL="$BASE" ;;
  *) START_URL="$WARMUP" ;;
esac

mkdir -p "$PROFILE"
echo "CDP port: $PORT"
echo "Profile:  $PROFILE"
echo "Start:    $START_URL"
echo "Once Perplexity looks normal (compose box visible), keep this Chrome open and use the MCP."

exec "$CHROME" \
  --remote-debugging-port="$PORT" \
  --user-data-dir="$PROFILE" \
  --no-first-run \
  --no-default-browser-check \
  --disable-features=Translate \
  "$START_URL"
