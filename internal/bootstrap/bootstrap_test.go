package bootstrap

import (
	"errors"
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
	base := filepath.Base(root)

	_, err := Init(Options{
		RepoRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}

	persona := filepath.Join(root, ".cursor", "perplexity", "project-persona.md")
	pack := filepath.Join(root, ".cursor", "perplexity", "packs", "deep-research.md")
	for _, want := range []string{persona, pack} {
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}

	skill := filepath.Join(root, ".cursor", "skills", slugify(base)+"-perplexity-research", "SKILL.md")
	if _, err := os.Stat(skill); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected skill copy at %s", skill)
	}

	body, err := os.ReadFile(persona)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if strings.HasPrefix(s, "---\nname:") {
		t.Error("persona must not get SKILL frontmatter")
	}
	if !strings.Contains(s, base) && !strings.Contains(s, "{{") {
		t.Errorf("persona should name the project, got %q", s[:min(120, len(s))])
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
