package perplexity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

func TestFormatMarkdown(t *testing.T) {
	md := perplexity.FormatMarkdown(perplexity.Conversation{
		URL:       "https://www.perplexity.ai/search/abc",
		ThreadID:  "thr_test",
		User:      "what is 2+2?",
		Assistant: "4",
		Citations: []result.Citation{{Title: "Math", URL: "https://example.com"}},
		Turns:     2,
	})
	for _, want := range []string{
		"# Perplexity export",
		"Thread: thr_test",
		"## Turn 1 (user)",
		"what is 2+2?",
		"## Turn 2 (assistant)",
		"4",
		"[Math](https://example.com)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
}

func TestWriteExportFile(t *testing.T) {
	dir := t.TempDir()
	path, err := perplexity.WriteExportFile(dir, "thr_abc", "# hi\n")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "# hi\n" {
		t.Fatalf("content %q", b)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Fatalf("expected .md path, got %s", path)
	}
}

func TestEnsureMarkdownExt(t *testing.T) {
	cases := map[string]string{
		"":                                      "export.md",
		"report.md":                             "report.md",
		"report.markdown":                       "report.markdown",
		"report.txt":                            "report.md",
		"79a0f5ef-7f85-4f69-b1f4-cfa3fed7832f": "79a0f5ef-7f85-4f69-b1f4-cfa3fed7832f.md",
	}
	for in, want := range cases {
		if got := perplexity.EnsureMarkdownExt(in); got != want {
			t.Fatalf("EnsureMarkdownExt(%q)=%q want %q", in, got, want)
		}
	}
}

func TestExportTargetPath(t *testing.T) {
	p := perplexity.ExportTargetPath("/tmp/exports", "thr_abc", "deep-research")
	if !strings.Contains(p, "thr_abc") {
		t.Fatalf("missing thread id: %s", p)
	}
	if !strings.HasSuffix(p, ".md") {
		t.Fatalf("missing .md: %s", p)
	}
	if !strings.Contains(p, "deep-research") {
		t.Fatalf("missing label: %s", p)
	}
}

func TestNewestExportAcceptsGUIDFile(t *testing.T) {
	dir := t.TempDir()
	guid := filepath.Join(dir, "ae79e6fc-1ffc-469a-949c-66d4b7bc3e7e")
	if err := os.WriteFile(guid, []byte("## Executive Options\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Touch mtime into the 2-minute window (WriteFile already recent).
	_ = os.Chtimes(guid, time.Now(), time.Now())

	path, err := perplexity.NormalizeExportFile(guid, dir, "thr_x", "poll")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Fatalf("normalized path should be .md: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Executive Options") {
		t.Fatalf("body missing: %s", b)
	}
}
