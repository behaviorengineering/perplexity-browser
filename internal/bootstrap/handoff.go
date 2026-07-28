package bootstrap

import (
	"fmt"
	"io"
	"strings"
)

// WriteHandoff prints human-oriented next steps after a successful init.
func WriteHandoff(w io.Writer, res Result) {
	if w == nil {
		return
	}
	workflowHint := res.PackRel
	if idx := strings.LastIndex(res.PackRel, "/"); idx >= 0 {
		workflowHint = res.PackRel[:idx]
	}
	if res.Layout == LayoutCursor {
		workflowHint = ".cursor/skills/" + res.WorkflowID + "/SKILL.md"
	} else {
		workflowHint = "docs/perplexity/workflow.md"
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Perplexity workflow scaffolded for %q (%s layout).\n\n", res.ProjectName, res.Layout)
	fmt.Fprintln(w, "Wrote:")
	for _, f := range res.Files {
		fmt.Fprintf(w, "  %s\n", f)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintln(w, "  1. MCP client: point perplexity-browser at perplexity-browser-mcp (headed browser).")
	fmt.Fprintln(w, "     See provider README: build, Cursor MCP config, PERPLEXITY_BROWSER_* env.")
	fmt.Fprintln(w, "  2. Login once: call perplexity_session (action=status). Sign in in the browser if need_login.")
	fmt.Fprintln(w, "  3. Customize: edit the pack question and constraints in:")
	fmt.Fprintf(w, "       %s\n", res.PackRel)
	fmt.Fprintf(w, "     Optional: tune triggers and guardrails in %s\n", workflowHint)
	fmt.Fprintln(w, "  4. Commit the generated files to this repo.")
	fmt.Fprintln(w, "  5. Research: ask your agent to run Perplexity research.")
	fmt.Fprintln(w, "     It should read the pack, show the full prompt, then perplexity_research (one shot) and export.")
	fmt.Fprintln(w, "     Perplexity output is research input only; validate before promoting to docs or code.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Workflow id: %s\n", res.WorkflowID)
	fmt.Fprintf(w, "Repo root:   %s\n", res.RepoRoot)
}
