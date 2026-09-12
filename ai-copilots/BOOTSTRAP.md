# BOOTSTRAP — perplexity-browser ai-copilots

**Module path:** `github.com/behaviorengineering/perplexity-browser`

```bash
MOD="$(go list -m -f '{{.Dir}}' github.com/behaviorengineering/perplexity-browser)"
mkdir -p .cursor/skills
ln -snf "$MOD/ai-copilots/skills/perplexity-browser-ops" .cursor/skills/perplexity-browser-ops
ln -snf "$MOD/ai-copilots/skills/perplexity-browser-research" .cursor/skills/perplexity-browser-research
```

Host overlays stay under `.cursor/perplexity/` (not inside the skill symlink).

```bash
test -f .cursor/skills/perplexity-browser-research/SKILL.md
```
