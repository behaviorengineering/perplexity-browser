package perplexity_test

import (
	"errors"
	"fmt"
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

	t.Run("assistant_only", func(t *testing.T) {
		md := perplexity.FormatMarkdown(perplexity.Conversation{
			URL:       "https://www.perplexity.ai/search/abc",
			ThreadID:  "thr_test",
			Assistant: "First paragraph.\n\nSecond paragraph.",
			Turns:     1,
		})
		if !strings.Contains(md, "## Turn 1 (assistant)") {
			t.Fatalf("expected assistant turn first:\n%s", md)
		}
		if strings.Contains(md, "(user)") {
			t.Fatalf("must not invent user turn:\n%s", md)
		}
	})
}

func TestConversationFromTexts(t *testing.T) {
	t.Parallel()
	ans := "First paragraph.\n\nSecond paragraph."
	conv, err := perplexity.ConversationFromTexts(
		"https://www.perplexity.ai/search/x",
		"thr_x",
		"",
		ans,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if conv.User != "" {
		t.Fatalf("user should be empty, got %q", conv.User)
	}
	if conv.Assistant != ans {
		t.Fatalf("assistant=%q want %q", conv.Assistant, ans)
	}
	if conv.Turns != 1 {
		t.Fatalf("turns=%d want 1", conv.Turns)
	}

	conv, err = perplexity.ConversationFromTexts("", "thr", "First paragraph.", ans, nil)
	if err != nil {
		t.Fatal(err)
	}
	if conv.User != "" {
		t.Fatalf("prefix user should be cleared, got %q", conv.User)
	}
}

func TestPromptTextSelectors(t *testing.T) {
	t.Parallel()
	ans := make(map[string]bool)
	for _, s := range perplexity.AnswerTextSelectors {
		ans[s] = true
	}
	var sawBubble bool
	for _, s := range perplexity.PromptTextSelectors {
		if ans[s] {
			t.Fatalf("prompt selector %q overlaps answer selectors", s)
		}
		if s == `[class*="user-bubble"]` {
			sawBubble = true
		}
	}
	if !sawBubble {
		t.Fatal(`PromptTextSelectors missing [class*="user-bubble"]`)
	}
}

func TestMenuDownloadError(t *testing.T) {
	t.Parallel()
	timeout := errors.New("timeout")
	err := perplexity.MenuDownloadError(
		"no more-menu download completed",
		`button[aria-label*="More options" i]`,
		[]error{
			fmt.Errorf(`(?i)export\s+as\s+markdown: not found`),
			fmt.Errorf(`(?i)copy\s+as\s+markdown: not visible: %w`, timeout),
		},
	)
	msg := err.Error()
	for _, want := range []string{
		"no more-menu download completed",
		`opener=button[aria-label*="More options" i]`,
		`(?i)export\s+as\s+markdown: not found`,
		`(?i)copy\s+as\s+markdown: not visible:`,
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
	if !errors.Is(err, timeout) {
		t.Fatalf("wrapped timeout not reachable via errors.Is: %v", err)
	}
	if perplexity.CodeOf(err) != perplexity.CodeUnavailable {
		t.Fatalf("code %q", perplexity.CodeOf(err))
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
		"":                                     "export.md",
		"report.md":                            "report.md",
		"report.markdown":                      "report.markdown",
		"report.txt":                           "report.md",
		"report.pdf":                           "report.pdf",
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

func TestIsPDFBytes(t *testing.T) {
	if !perplexity.IsPDFBytes([]byte("%PDF-1.4\n%")) {
		t.Fatal("expected PDF magic")
	}
	if perplexity.IsPDFBytes([]byte("# Perplexity export\n")) {
		t.Fatal("markdown is not PDF")
	}
	if !perplexity.LooksLikePDFName("thread.pdf") {
		t.Fatal("expected .pdf name")
	}
}

func TestNormalizeExportFileRefusesPDF(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "print.pdf")
	if err := os.WriteFile(src, []byte("%PDF-1.4\n% Chromium\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := perplexity.NormalizeExportFile(src, dir, "thr_x", "poll"); err == nil {
		t.Fatal("expected refuse PDF")
	}
}

func TestExportMenuPatternsRejectBareDownload(t *testing.T) {
	for _, sel := range append(append([]string{}, perplexity.ExportMenuPatterns...), perplexity.DeepResearchDownloadPatterns...) {
		if sel == `(?i)^download$` || sel == `(?i)^export$` {
			t.Fatalf("bare %q matches Print/PDF", sel)
		}
	}
}
