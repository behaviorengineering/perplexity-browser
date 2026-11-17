# Cursor and Claude plugins

Fleet install for **perplexity-browser-research**, **perplexity-browser-ops**, and the **perplexity-browser** MCP server (stdio via `scripts/mcp-serve.sh`, which always runs `perplexity-browser-mcp serve`).

Manifests: `.cursor-plugin/` (Cursor), `.claude-plugin/` (Claude Code). MCP: `mcp.json` (Cursor), `.mcp.json` (Claude).

## Local development (maintain on this machine)

Cursor indexes a **real directory** under `~/.cursor/plugins/local/`. It does **not** load a symlink whose target is outside that folder.

```bash
git clone git@github.com:behaviorengineering/perplexity-browser.git ~/.cursor/plugins/local/perplexity-browser
cd ~/.cursor/plugins/local/perplexity-browser
make build
```

Alternatively, from an existing clone:

```bash
git worktree add ~/.cursor/plugins/local/perplexity-browser feat/cursor-claude-plugins
cd ~/.cursor/plugins/local/perplexity-browser
make build
```

Workflow:

1. Edit in that directory, open a PR on `github.com/behaviorengineering/perplexity-browser`
2. `git pull` after merge
3. **Developer: Reload Window**
4. Start a **new** Agent conversation

## Team marketplace

1. Dashboard or Claude: import marketplace from `https://github.com/behaviorengineering/perplexity-browser`
2. Install plugin `perplexity-browser`
3. From the installed plugin directory (or after clone): `make build` or `make install` so `bin/perplexity-browser-mcp` exists for the wrapper

## Dual-run warning (user-level MCP)

If Cursor still has a **user-level** `perplexity-browser` or `user-perplexity-browser` MCP entry **and** the plugin MCP is enabled, tools duplicate and two processes may fight over `~/.perplexity-browser-mcp/profile`.

**After the plugin works:** disable the user-level MCP entry. Keep one server only.

## Host overlay (consumer repo)

Plugin ships library skills. Each host keeps **real files** (not symlinks into the plugin):

```text
.cursor/perplexity/
  project-persona.md
  packs/
    deep-research.md
    prose-review.md
```

Scaffold a new host:

```bash
perplexity-browser-mcp init .
```

Do not duplicate library research skills into the host `.cursor/skills/` tree. Use the plugin or [ai-copilots/BOOTSTRAP.md](../ai-copilots/BOOTSTRAP.md) for skill discovery.

## Claude Code

Install from marketplace or:

```bash
claude plugin validate --plugin-dir /path/to/perplexity-browser
```

Skills load from `./ai-copilots/skills/`; MCP from `.mcp.json`.

## Fallback without plugin

Go module consumers: [ai-copilots/BOOTSTRAP.md](../ai-copilots/BOOTSTRAP.md) (`ln -snf` into `.cursor/skills/`). Use when the plugin is unavailable (some agents, Copilot, Codex).