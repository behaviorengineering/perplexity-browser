package bootstrap

import (
	"strings"
	"testing"
)

func TestWriteHandoff(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	WriteHandoff(&b, Result{
		Files:       []string{"/tmp/repo/.cursor/perplexity/project-persona.md"},
		ProjectName: "foo",
		WorkflowID:  "foo-perplexity-research",
		PackRel:     ".cursor/perplexity/packs/deep-research.md",
		Layout:      LayoutCursor,
		RepoRoot:    "/tmp/repo",
	})
	out := b.String()
	for _, want := range []string{
		"Next steps:",
		"perplexity_session",
		"deep-research.md",
		"project-persona.md",
		"research input only",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("handoff missing %q:\n%s", want, out)
		}
	}
}
