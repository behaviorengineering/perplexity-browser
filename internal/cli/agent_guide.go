package cli

import "fmt"

func agentOperatingGuide() string {
	return fmt.Sprintf(`perplexity-browser-mcp %s — headed Playwright MCP for Perplexity Pro

ROLE & BOUNDARIES
  Exposes MCP tools over stdio; requires logged-in Chrome user data dir.
  Does not start on bare invocation; use explicit serve for the MCP transport.

AGENT OPERATING GUIDE
  Read AGENTS.md and ai-copilots/skills/perplexity-browser-ops/SKILL.md before build/smoke.
  Research workflow: ai-copilots/skills/perplexity-browser-research/SKILL.md

COMMANDS BY RISK & LIFECYCLE
  Inspect & Validate
    version      Build identity
    help         Flag reference

  Execute & Mutate
    serve        Start MCP stdio server (default for Cursor wiring)
    init         Scaffold workflow files in a host repo

AUTOMATION RULES FOR AGENTS
  - Never commit Perplexity session cookies or user-data paths into git.
  - Unknown commands exit non-zero; bare invoke exits 0 with this guide.
`, Version)
}
