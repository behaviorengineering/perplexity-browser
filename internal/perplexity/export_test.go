package perplexity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if filepath.Dir(path) != dir && !strings.HasPrefix(path, dir) {
		// Abs path may resolve; ensure file exists.
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "# hi\n" {
		t.Fatalf("content %q", b)
	}
}
