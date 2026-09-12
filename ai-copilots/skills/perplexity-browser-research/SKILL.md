---
name: perplexity-browser-research
description: >-
  Run Perplexity Pro research via the Perplexity Browser MCP
  (user-perplexity-browser). Use when the user says open in Perplexity,
  perplexity research, Research mode, Deep Research, paste pack, doctrine
  lookup, or export perplexity chat. Deep research uses compose `/` modality
  menu. Browser output is research input only; never treat as project fact
  without validation.
---

# Perplexity browser research

**Why:** Drive **Perplexity Pro** through the **Perplexity Browser MCP** (`user-perplexity-browser`): headed login, Deep Research, continue, export.

**Not a source of truth.** Export is research input. Fact-check against primary sources and the project's own doctrine or content rules before writing repo files.

## Defaults in this skill

| Path | Role |
|------|------|
| `reference/project-persona.md` | Fallback voice and risk frame when the consumer has no overlay |
| `packs/deep-research.md` | Default fact / grounding prompt template (`mode=deep`) |
| `packs/prose-review.md` | Default human-voice / AI-artifact cold-read (`mode=search`) |

**Project overlay (consumer-owned, never inside this skill folder):**

```
.cursor/perplexity/
  project-persona.md
  packs/
    deep-research.md
    prose-review.md
    <optional extra packs>
```

**MUST NOT** write overlay files under `.cursor/skills/perplexity-browser-research/` (that path is a pack symlink; edits land in cursor-packs and pack bumps overwrite them).

**Load order for persona and packs:**

1. If `.cursor/perplexity/project-persona.md` exists, Read it; else Read this skill's `reference/project-persona.md`.
2. If `.cursor/perplexity/packs/<name>.md` exists for the chosen workflow, fill that; else fill this skill's `packs/<name>.md`.
3. One pack per run.

**MUST** Read the chosen persona before filling a pack.

## When to invoke

**Natural triggers:** "Let's open this in Perplexity", "Research mode", `perplexity export <topic>`.

**Workflow triggers:** doctrine or literature lookup, paste pack, research before drafting, product overlay polish phases that call for Perplexity.

## Priority order

| Order | Tool | When |
|-------|------|------|
| 1 | **Perplexity Browser MCP** | Default: `perplexity_research` / `_continue` / `_export` / `_session` |
| 2 | **Paste pack** → human runs manually | Login/2FA blocked, or user wants full control |
| 3 | **Playwright MCP** (`user-playwright`) | Only if browser MCP fails with `ui_changed` or equivalent selector break |
| 4 | **Perplexity API MCP** | Last resort; thinner than Pro Deep research |

## MCP workflow (assistant)

Discover schemas with `GetDynamicTools` (or the host's MCP schema tool) on `user-perplexity-browser` before calling.

### 1. Prepare prompt

- Read `.cursor/perplexity/project-persona.md` if it exists; else this skill's `reference/project-persona.md`.
- Fill **one** pack: `.cursor/perplexity/packs/<name>.md` if it exists; else this skill's `packs/<name>.md` (`deep-research` or `prose-review`).
- **No sensitive identifiers in `title_hint`** (employer brand, real names, client labels). Facts in the prompt body are the user's risk choice.
- Keep the prompt path in chat so the user can re-open it.

### 2. Session

```text
perplexity_session  action=status  session_id=<optional>
```

If not logged in: tell the user to complete sign-in in the headed window, then:

```text
perplexity_session  action=wait_for_login
```

With `PERPLEXITY_BROWSER_CDP_URL` set, the MCP auto-launches Chrome when CDP is down. No manual script step unless auto-launch is disabled or the profile is locked by another Chrome.

Other actions: `close` (stops window; keeps profile), `cancel` (abort in-flight wait).

### 3. Research

```text
perplexity_research
  prompt:     <full pack / prepared text>
  mode:       deep          # doctrine / literature / long research; use search only for short polish
  title_hint: <optional short label, no sensitive IDs — thread history hygiene>
  session_id: <optional override; default is this workspace folder name>
  timeout_ms: <optional; Deep research can take many minutes>
```

**How Deep Research is selected (Perplexity UI):** compose uses a **`/` modality command menu**, not only the old Search pill.

| Step | UI action (MCP automates when `mode=deep`) |
|------|--------------------------------------------|
| 1 | Focus the ask/compose box |
| 2 | Type `/` → command menu (Deep Research, Model Council, Plan mode, Skills, …) |
| 3 | Choose **Deep Research** (telescope); click **Use** if the detail pane shows it |
| 4 | Paste/submit the prepared prompt |

`mode=search` leaves the default Search compose (no `/` Deep Research arming). Prefer `mode=deep` for doctrine and long research; prefer `mode=search` for the prose-review pack.

If automation reports `ui_changed` on mode select, the user can arm Deep Research manually with `/` then tell the assistant to retry submit (or use paste pack).

After MCP restart, `continue` / `export` reload the thread for the same **session scope** from `~/.perplexity-browser-mcp/sessions/<scope>.json`.

**Global Cursor MCP:** you do **not** need a per-repo env var. Scope defaults to the **workspace folder name**. Only pass `session_id` to override. Same scope on every tool call in one thread (or omit it).

### 4. Follow-up (optional)

```text
perplexity_continue  message=<follow-up>  thread_id=<optional; default active>
```

### 5. Export

```text
perplexity_export  format=markdown  save_dir=<optional>
```

Default export dir is `PERPLEXITY_BROWSER_EXPORT_DIR` (often `~/.perplexity-browser-mcp/exports`).

Export order:

1. **UI download** — ⋯ (three-dots) menu → Export as Markdown, and Deep Research download controls via Playwright `ExpectDownload` + `SaveAs` (handles CDP GUID filenames without `.md`)
2. **Scrape fallback** — visible thread answer written as markdown (`method=scrape`; may be a Deep Research *summary*, not the full report file)
3. **`export_manual`** — only if both fail

**If `status` is `export_manual`:** tell the user to use ⋯ → Export as Markdown (or Deep Research Download) in the headed browser, then paste or save the file. Include the thread `url`.

**If `method` is `scrape`:** say so; offer a manual Deep Research download when the full report file is needed.

**Do not** retry `perplexity_export` in a loop hoping for a different result; one automated attempt, then human export if still incomplete.

### 6. After import

- Follow the persona's validate-before-write rules
- Validate against primary sources and project doctrine or the target post
- Reject blog-only citations presented as binding authority
- **MUST NOT** paste Perplexity prose into ship paths without the project's normal review or drafting skill
- Optional: append verified bullets to project notes only after the user confirms

## MCP tools

| Tool | Use |
|------|-----|
| `perplexity_session` | Login status, wait for human login, close, cancel |
| `perplexity_research` | New thread; `deep` or `search`; submit; wait; extract |
| `perplexity_continue` | Follow-up on active (or named) thread |
| `perplexity_export` | Share UI → markdown file; `export_manual` → ask user to Share/copy |

Provider pitch when vendored: `providers/perplexity-browser/README.md`.

## Discuss note template

For non-trivial research, create a discuss note (project path; for example `docs/research/` or `pii/drafts/discuss/`) with:

- Pack / prompt text
- Thread id or URL after submit
- Export path or import instructions for the user

## Limits

- UI changes can break the MCP; fall back to paste pack or Playwright only when the MCP reports failure
- Sensitive data in cloud Perplexity threads: the user's risk choice
- Automation is for personal Pro use; no credential harvesting

## Related

- Defaults live in this skill's `reference/` and `packs/`
- Project extensions live in `.cursor/perplexity/` (real files in the consumer repo)
- MCP server: `user-perplexity-browser`
