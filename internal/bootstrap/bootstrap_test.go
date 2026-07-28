package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitDocsLayout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("# test\n"), 0o644)

	res, err := Init(Options{
		RepoRoot:    root,
		Layout:      LayoutDocs,
		ProjectName: "Acme API",
		PackSlug:    "deep-research",
		Mode:        "deep",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 2 {
		t.Fatalf("files: %v", res.Files)
	}

	workflow := filepath.Join(root, "docs", "perplexity", "workflow.md")
	pack := filepath.Join(root, "docs", "perplexity", "packs", "deep-research.md")
	for _, want := range []string{workflow, pack} {
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}

	body, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if strings.Contains(s, "{{") {
		t.Errorf("unreplaced placeholders in workflow:\n%s", s)
	}
	if !strings.Contains(s, "Acme API") {
		t.Error("missing project name")
	}
	if !strings.Contains(s, "`README.md`") {
		t.Error("missing discovered prep path")
	}
}

func TestInitCursorLayoutDefault(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	_, err := Init(Options{
		RepoRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}

	skill := filepath.Join(root, ".cursor", "skills", slugify(filepath.Base(root))+"-perplexity-research", "SKILL.md")
	body, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.HasPrefix(s, "---\nname:") {
		t.Errorf("missing cursor frontmatter: %q", s[:min(80, len(s))])
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	opts := Options{RepoRoot: root}
	if _, err := Init(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(opts); err == nil {
		t.Fatal("expected error on second init without force")
	}
}

func TestInitForceOverwrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	opts := Options{RepoRoot: root, Force: true}
	if _, err := Init(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(opts); err != nil {
		t.Fatal(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
