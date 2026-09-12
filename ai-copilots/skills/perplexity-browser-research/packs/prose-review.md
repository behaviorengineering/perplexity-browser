# Pack — Prose review (default)

Workflow: `perplexity-browser-research`. Mode: **search** (default). Use **deep** only for a very long draft.

**Not fact-checking.** Judge readability, flow, voice, and AI artifacts only.

Fill every `[bracket]` before submit. Copy the **whole** block into `perplexity_research` `prompt`.

Read **`.cursor/perplexity/project-persona.md`** if it exists; else **`.cursor/skills/perplexity-browser-research/reference/project-persona.md`**.

---

## Context

Editorial cold-read for **`[project name]`**.

**Repo / workspace:** `[path or remote]`

**Target:** `[path — title]`

**Scope** (pick one; delete others):

- [ ] **Full piece**
- [ ] **Lead / abstract only**
- [ ] **Body only**
- [ ] **Selection:** `[paragraph range]`

**Reader persona:** `[who skims this; what makes them bounce]`

**After this review:** A human applies fixes via the project's revise skill. Perplexity diagnoses; it does not replace those passes.

---

## Question

How well does this prose read for **humans**? Where does it **lose trust, rhythm, or clarity**? Where does it sound **AI-generated, teacherly, or over-polished**?

**What I need from you:**

1. **Cold-read verdict** in one paragraph
2. **Quoted failures only** (exact phrases)
3. **What already works** (2–5 lines)
4. **Top 5 fixes** ranked by impact
5. **Do not rewrite the whole piece**; give direction, not a full replacement draft

---

## Constraints

**Your role**

- Act as a **skeptical editor**, not a writing coach and not a fact-checker.
- Penalize prose that is smooth but empty.
- Penalize symmetrical punch-line rhythm.
- Reward named subjects, concrete mechanisms, and one clear metaphor thread.
- US English unless the project says otherwise. No em dashes (U+2014) in suggested rewrites when the project bans them.

**AI artifacts to flag** (quote when found)

| Pattern | Example shape |
| --- | --- |
| Teacher / editor framing | "You will learn...", "The useful move is..." |
| Bridge verbs | "bridges X and Y", "ties together" |
| Semicolon aphorisms | "Stress drains; purpose concentrates." |
| Negation-first fluff | "It is not X; it is Y" |
| Meta-commentary | "This makes the point clear", "The pattern matters" |
| Hedge piles | "One possibility is...", "It is important to note..." |
| Staccato thesis stacks | Same-length punch lines every sentence |

**Standalone worth test:** If this sentence stood alone, would it be worth saying?

**Out of scope**

- Factual accuracy and citation quality (use `deep-research.md`)
- SEO and formatting tooling
- Translation siblings unless the project asks

**Limitation:** You are also an LLM. Prefer quoted evidence over confidence.

---

## Repo excerpts (paste below)

### Excerpt A — Lead / abstract (if in scope)

```text
[paste]
```

### Excerpt B — Body (if in scope)

```text
[paste]
```

### Excerpt C — Author note (optional)

```text
[voice target, metaphors to keep]
```

---

## Output format (request from Perplexity)

Please structure your answer as:

1. **Cold-read verdict**
2. **Scores** (1–5): Clarity | Flow | Human voice | Stakes | Trust
3. **Keep** (quoted lines that work)
4. **AI artifact table**: Quote | Location | Pattern | Why it fails | Fix direction
5. **Flow map**
6. **Top 5 fixes**
7. **Paragraphs to cut or merge**
8. **Open questions** for the author

---

## After export (repo handling)

1. Export via `perplexity_export`.
2. Optional note with verdict, top fixes, thread URL.
3. Apply edits with the project's voice / flow revise skills. Do not paste wholesale rewrites.
4. Re-run only if the draft changed heavily.
