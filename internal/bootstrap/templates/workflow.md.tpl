# {{PROJECT_NAME}} — Perplexity research workflow

**Repo:** {{PROJECT_NAME}}  
**Workflow id:** {{WORKFLOW_ID}}

This document tells the **caller** (human or agent) how to run **Perplexity Pro** via the **Perplexity Browser MCP** for this repository: what to prepare, which tools to call, and where results go.

Perplexity output is **research input only**. {{VALIDATION_SUMMARY}}

## When to use

{{TRIGGER_BULLETS}}

## Workflows

| Workflow | Pack | Mode | When |
|----------|------|------|------|
{{WORKFLOWS_TABLE}}

## Tool priority

| Order | Method | When |
|-------|--------|------|
| 1 | **Perplexity Browser MCP** | `perplexity_research`, `perplexity_continue`, `perplexity_export`, `perplexity_session` |
| 2 | **Manual paste pack** | Login blocked, or human runs Perplexity in browser |
| 3 | **Generic browser automation** | Only if MCP returns `ui_changed` or equivalent |

Discover MCP tool schemas through your client's introspection before calling.

## Procedure (caller)

### 1. Prepare prompt

Read these repo paths first (source of truth):

{{PREP_PATHS}}

Fill the matching pack:

{{PACK_LIST}}

Rules:

- Show the **full prepared prompt** to the human before `perplexity_research` when they are in the loop.
- Do **not** put secrets or customer identifiers in `title_hint`.
- Prompt body and cloud policy: {{CLOUD_PROMPT_POLICY}}

### 2. Session

```text
perplexity_session  action=status
```

If not logged in: human signs in in the headed window, then `perplexity_session` `action=wait_for_login`.

### 3. Research

```text
perplexity_research
  prompt:     <entire filled pack>
  mode:       deep | search
  title_hint: <short label, no secrets>
  session_id: <optional; default is often project folder name>
  timeout_ms: <optional>
```

Submit the **whole** pack in one `prompt`. Do not drip-feed the first submit through `perplexity_continue`.

### 4. Follow-up (optional)

```text
perplexity_continue  message=<follow-up>  thread_id=<optional>
```

Only after the initial `perplexity_research`.

### 5. Export

```text
perplexity_export  format=markdown  save_dir=<optional>
```

Default export dir: `PERPLEXITY_BROWSER_EXPORT_DIR` (often `~/.perplexity-browser-mcp/exports`).

If status is `export_manual`: share thread URL; one automated export attempt only.

### 6. After import

{{EXPORT_SINK}}

{{GUARDRAILS}}

## Research note (optional)

For non-trivial runs, create `{{RESEARCH_NOTE_PATH}}` with:

- Pack used (path) and prompt snapshot or link
- Thread URL / id
- Export file path
- Summary bullets and open questions

## Limits

- UI changes can break automation; fall back to manual paste when MCP fails
- Perplexity is not authoritative for this repo until validation rules pass
- Personal Pro use only; no credential harvesting

## Bootstrap

Regenerate scaffold: `perplexity-browser-mcp init --force` from this repo root.

MCP install: [perplexity-browser README](https://github.com/behaviorengineering/perplexity-browser).
