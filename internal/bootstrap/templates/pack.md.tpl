# Pack — {{PACK_TITLE}}

Workflow: `{{WORKFLOW_ID}}`. Mode: **{{MODE}}** (`deep` or `search`).

Fill every section from repo files before submit. Copy the whole block into `perplexity_research` `prompt`.

---

## Context

{{CONTEXT_DESCRIPTION}}

**Repo:** {{PROJECT_NAME}}

**Relevant paths already reviewed:**

{{REVIEWED_PATHS}}

---

## Question

{{PRIMARY_QUESTION}}

---

## Constraints

{{CONSTRAINTS}}

- Do not invent facts about this codebase; flag gaps explicitly.
- Prefer primary sources, official docs, and recent material.
- {{EXTRA_CONSTRAINTS}}

---

## Repo excerpts (paste below)

### Excerpt A — {{EXCERPT_A_LABEL}}

```text
[paste from {{EXCERPT_A_PATH}}]
```

### Excerpt B — {{EXCERPT_B_LABEL}} (optional)

```text
[paste from {{EXCERPT_B_PATH}}]
```

---

## Output format (request from Perplexity)

Please structure your answer as:

1. **Summary** (5–10 bullets)
2. **Recommendation** (if applicable)
3. **Risks / tradeoffs**
4. **Sources** (links; note if blog-only vs primary)
5. **Open questions** for the team

---

## After export (repo handling)

{{POST_EXPORT_INSTRUCTIONS}}
