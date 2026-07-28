# Repo integration (Perplexity Browser MCP)

**Audience:** humans and **any** MCP-capable agent (Cursor, Claude Desktop, custom runners). Not tied to one IDE.

**Goal:** One-time bootstrap per consumer repo so the **caller** (your agent) knows how to prepare prompts, call MCP tools, and file exports. The MCP server stays unchanged.

## Layers

| Layer | Owns | Lives in |
|-------|------|----------|
| **MCP server** | Browser, login persistence, submit, wait, continue, export | Global install (`bin/perplexity-browser-mcp`) |
| **Repo workflow** | Triggers, prep paths, paste packs, export sink, guardrails | Consumer repo (see below) |
| **Human** | MCP install, Pro login / 2FA, approve generated files, validate imports | — |

```text
Human installs MCP + logs in once
        │
        ▼
perplexity-browser-mcp init [repo]   (default: .cursor/skills/<folder>-perplexity-research/)
        │
        ▼
SKILL.md + packs/ in consumer repo
        │
        ▼
Day to day: agent reads workflow → fills pack → perplexity_research(full prompt) → export
```

Default `init` writes **Cursor skill** layout (`.cursor/skills/<folder>-perplexity-research/`). Use `--layout docs` for `docs/perplexity/` instead.

## Where generated files go

Pick **one** layout per repo (agent asks if unclear):

| Layout | Workflow doc | Packs | Best for |
|--------|--------------|-------|----------|
| **Cursor (default)** | `.cursor/skills/<workflow-id>/SKILL.md` | same folder `packs/` | Cursor (YAML frontmatter included) |
| **Docs** | `docs/perplexity/workflow.md` | `docs/perplexity/packs/*.md` | Any client; versioned in git |
| **Minimal** | `docs/perplexity.md` | inline sections in same file | Tiny repos, one workflow (manual) |

Commit generated files to the consumer repo. Re-run bootstrap only when adding workflows or refreshing triggers.

## Preconditions (human)

1. Build and register the MCP server in your client (see [README](../README.md)).
2. `perplexity_session` with `action=status`. If `need_login`, sign in in the headed browser, then `action=wait_for_login`.
3. Scaffold workflow files in the consumer repo (see **Quick start** below).

Agents: discover tool schemas through your client's MCP introspection before calling tools. Tool names are stable: `perplexity_session`, `perplexity_research`, `perplexity_continue`, `perplexity_export`.

## Quick start (binary)

From a built `perplexity-browser-mcp` binary:

```bash
perplexity-browser-mcp init /path/to/consumer-repo
perplexity-browser-mcp init .
perplexity-browser-mcp init . --layout docs
perplexity-browser-mcp init . --force   # overwrite existing scaffold
```

**Default output (cursor layout):**

```text
.cursor/skills/<folder>-perplexity-research/SKILL.md
.cursor/skills/<folder>-perplexity-research/packs/deep-research.md
```

**Docs layout** (`--layout docs`):

```text
docs/perplexity/workflow.md
docs/perplexity/packs/deep-research.md
```

The binary embeds templates from `internal/bootstrap/templates/`. It discovers prep paths (`README.md`, `docs/`, etc.) when they exist. Customize placeholders in the generated files after init, or re-run with `--force` after editing templates in the provider (for maintainers).

**Flags:** `--project`, `--workflow`, `--layout`, `--pack`, `--mode`, `--force`. Run `perplexity-browser-mcp help` for details.

## Bootstrap procedure (agent, optional customization)

Run **once** per repo unless extending workflows. Prefer **`perplexity-browser-mcp init`** first; use the steps below to customize packs, guardrails, and extra workflows after scaffold.

### Phase 0 — Confirm scope

State in chat:

1. Consumer repo root and default `session_id` (often workspace / project folder name; override with `session_id` on tools if two repos share the same basename).
2. Whether a workflow doc already exists at the chosen layout (extend vs replace; **ask before overwrite**).
3. Goal: first-time setup | add pack | refresh triggers.

### Phase 1 — Repo discovery (read before asking)

Scan (do not invent):

- `README.md`, `docs/`, `ARCHITECTURE.md`, `ADR*`, agent config dirs if present
- Draft or research paths (`docs/research/`, `notes/`, etc.)
- Secrets policy (`.env.example`, security docs)

Infer: project label, repo type, source-of-truth paths, 1–3 likely research jobs.

Ask the human only **gaps**:

1. Top **1–2** workflows to support first
2. **Export sink** (e.g. `docs/research/`, PR-only, no file write)
3. **Cloud prompt policy** (OK to paste customer names in prompt body? `title_hint` must never contain secrets)
4. **Validation** (what must not be auto-promoted without human yes)
5. **Artifact layout** (table above)

### Phase 2 — Name and paths

Default workflow id: `<project>-perplexity-research` (lowercase, hyphens).

Example (docs layout):

```text
docs/perplexity/workflow.md
docs/perplexity/packs/<slug>.md
```

### Phase 3 — Generate from templates

**Preferred:** run `perplexity-browser-mcp init` (see Quick start).

**Manual / extra packs:** templates embedded in the binary at `internal/bootstrap/templates/`:

| Template | Output |
|----------|--------|
| `workflow.md.tpl` | Consumer workflow doc |
| `pack.md.tpl` | One file per workflow under `packs/` |

Substitute all `{{PLACEHOLDER}}` tokens from Phase 1. Do not commit unreplaced placeholders.

| Placeholder | Content |
|-------------|---------|
| `{{WORKFLOWS_TABLE}}` | Rows: name, pack file, `deep`/`search`, when to use |
| `{{PACK_LIST}}` | Bullet paths to pack files |
| `{{PREP_PATHS}}` | Repo paths to read before filling packs |
| `{{EXPORT_SINK}}` | Where exports land and how to cite them |
| `{{GUARDRAILS}}` | Repo-specific must-not rules |
| `{{CLOUD_PROMPT_POLICY}}` | Human answer from Phase 1 |

Optional: one line in consumer `README.md` pointing at `docs/perplexity/workflow.md`.

### Phase 4 — Smoke test

1. `perplexity_session` `action=status`
2. Fill the **smallest** pack from real repo files
3. Show the **full prepared prompt** to the human before submit
4. `perplexity_research` with entire `prompt`, correct `mode`, `title_hint` without secrets
5. `perplexity_export` `format=markdown`
6. If `export_manual`: stop; share thread `url`; do not retry export in a loop
7. Summarize export path vs export sink

**Must not** use `perplexity_continue` for the first submit. **Must** use one full `perplexity_research` prompt.

### Phase 5 — Handoff

Tell the human:

- Paths to workflow doc and packs
- Phrases that load this workflow in future sessions
- Commit reminder
- Bootstrap complete; use generated workflow for normal runs

## Daily workflow (generated doc)

The generated `workflow.md` (from template) defines per-repo:

- When to run Perplexity
- Which pack to fill
- MCP call sequence
- Post-export validation

Core MCP sequence (all clients):

```text
perplexity_session     action=status
perplexity_research    prompt=<full pack>  mode=deep|search  title_hint=<no secrets>
perplexity_continue    message=...         # optional follow-ups only
perplexity_export      format=markdown
```

## Optional: docs layout in non-Cursor clients

If you use `--layout docs`, workflow lives under `docs/perplexity/`. Cursor users can still open that file manually or re-run `perplexity-browser-mcp init` without `--layout docs` to get the skill layout (use `--force` if files exist).

## Must / must not (bootstrap)

| Must | Must not |
|------|----------|
| Read the repo before generating | Copy domain-specific packs from other projects (e.g. legal/case packs) |
| Ask before overwrite | Add MCP server code during bootstrap |
| Show full prompt before smoke research | Drip-feed first submit via `perplexity_continue` |
| Treat Perplexity output as input only | Promote exports to source of truth without repo validation rules |

## Reference consumer

Consilium (case/legal-specific): workflow skill `perplexity-browser-research` and `perplexity-pack.md` under that repo's `.cursor/skills/`. Use as a **pattern**, not a copy-paste source.

## Related

- [README](../README.md) — build, env, MCP client config
- `perplexity-browser-mcp init` — scaffold consumer workflow files
- `internal/bootstrap/templates/` — embedded templates (source in provider repo)
