# Pack — Deep research (default)

Workflow: `perplexity-browser-research`. Mode: **deep** (use `search` only for a quick single-fact lookup).

Fill every `[bracket]` from repo files before submit. Copy the **whole** block into `perplexity_research` `prompt`.

Read **`.cursor/perplexity/project-persona.md`** if it exists; else **`.cursor/skills/perplexity-browser-research/reference/project-persona.md`**. Align voice and risk with it.

---

## Context

External research for **`[project name]`**.

**Repo / workspace:** `[path or remote]`

**Relevant paths reviewed before this pack:**

- Target artifact: `[path]`
- Doctrine / notes: `[path if any]`
- Prior research: `[path if any]`

---

## Research intent

Pick **one** primary intent (delete the others):

- [ ] **Fact-check** claims before publish or send
- [ ] **Grounding hunt** for citations and definitions
- [ ] **Source map** for a long interview, brief, or draft
- [ ] **Topic explore** before drafting

**Target (if any):** `[path, title, type]`

**Why now:** `[one sentence: what decision this research unlocks]`

---

## Question

[State the research question in plain English. Be specific.]

**Claims or hooks to validate** (one per line):

1. [claim]
2. [claim]
3. [claim]

**Optional follow-ups:**

- [best primary citation for each supported claim]
- [consensus vs speaker framework vs rhetoric]

---

## Constraints

**Evidence quality**

- Prefer peer-reviewed papers, official institutional pages, and primary docs over secondary blogs.
- Distinguish established fact, interpretive framework, and rhetoric.
- Do not invent facts about this codebase; flag gaps explicitly.
- Prefer recent material unless the question is historical.

**How findings will be used**

- `[project editorial or legal constraints; e.g. US English, no em dash, Source link format]`
- Perplexity output is **research input only**; a human validates before edits ship.

**Out of scope** (unless the question says otherwise)

- `[tooling / theme / translation / SEO — list project-specific outs]`
- Prose quality / AI voice (use `prose-review.md` instead)

---

## Repo excerpts (paste below)

### Excerpt A — Claims or list-facing copy

```text
[paste]
```

### Excerpt B — Body or draft spine

```text
[paste]
```

### Excerpt C — Sources already in hand (optional)

```text
[paste URLs, titles, IDs]
```

---

## Output format (request from Perplexity)

Please structure your answer as:

1. **Summary** (5–10 bullets)
2. **Claim-by-claim verdict table**: Claim | Verdict (supported / partial / unsupported / unclear) | Best primary source | Nuance
3. **Recommendation**: Keep | Soften | Cut or fix
4. **Grounding candidates** (if needed): title + URL + one-line digest
5. **Risks / tradeoffs**
6. **Sources** (label peer-reviewed vs preprint vs blog)
7. **Open questions**

---

## After export (repo handling)

1. Export via `perplexity_export`; note path.
2. Save a discuss or research note with pack path, thread URL, export path, summary, open questions.
3. After human review, apply edits via the project's drafting / revise skill.
4. Do **not** commit raw exports into ship paths without validation.
