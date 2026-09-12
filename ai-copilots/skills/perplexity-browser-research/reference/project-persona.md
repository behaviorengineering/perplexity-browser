# Project Perplexity persona (default)

Copy this file to **`.cursor/perplexity/project-persona.md`** in the consumer repo and fill the brackets. Do **not** copy it into `.cursor/skills/perplexity-browser-research/` (pack symlink). The pack skill MUST Read `.cursor/perplexity/project-persona.md` when present; otherwise use this default.

## Who we are

- **Project / product:** `[name]`
- **Audience:** `[who reads the output: editors, counsel, engineers, public site readers]`
- **What Perplexity is for here:** `[fact-check | literature map | cold-read prose | doctrine lookup | other]`

## Voice when writing prompts

- Prefer **concrete nouns** and named claims over vague “research this topic.”
- Ask for **primary sources** and study design when facts matter.
- Say what is **out of scope** (tooling, translation, SEO) so the model does not wander.
- **US English** in prompts and requested rewrites unless the project says otherwise.
- **MUST NOT** use the em dash character (U+2014) in suggested rewrites when the project bans it.

## Risk and privacy

- **Sensitive labels** (client names, employers, case IDs): keep out of `title_hint`; prompt body is the user's risk choice.
- Treat every export as **research input**, not shippable fact.
- **MUST NOT** paste Perplexity prose into publish paths without the project's normal review skill.

## Validate-before-write

1. Compare claims to primary sources or project doctrine.
2. Reject blog-only citations presented as binding authority.
3. Apply edits through the project's drafting / revise skill, not wholesale paste.

## Pack choice

| Need | Pack | Mode |
|------|------|------|
| Facts, sources, grounding | `.cursor/perplexity/packs/deep-research.md` if present, else this skill's `packs/deep-research.md` | deep |
| Human voice, AI artifacts | `.cursor/perplexity/packs/prose-review.md` if present, else this skill's `packs/prose-review.md` | search |

Consumers MAY add extra files under `.cursor/perplexity/packs/` (email, doctrine, claims-only). Prefer those when they exist for this workflow.
