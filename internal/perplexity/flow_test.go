package perplexity_test

import (
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
)

func TestTruncate(t *testing.T) {
	s, trunc := perplexity.Truncate("hello", 10)
	if trunc || s != "hello" {
		t.Fatalf("got %q trunc=%v", s, trunc)
	}
	s, trunc = perplexity.Truncate("abcdefghijklmnop", 5)
	if !trunc || !containsPrefix(s, "abcde") {
		t.Fatalf("got %q trunc=%v", s, trunc)
	}
}

func TestProjectDialogFieldPattern(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"Project description", true},
		{"Describe what this project is for", true},
		{"Ask anything…", false},
		{"New collection", true},
	}
	for _, tc := range cases {
		got := perplexity.ProjectDialogFieldPattern.MatchString(tc.s)
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.s, got, tc.want)
		}
	}
}

func TestNewThreadButtonNamesAvoidBareNew(t *testing.T) {
	for _, name := range perplexity.NewThreadButtonNames {
		if name == "New" {
			t.Fatal(`bare "New" must not be in NewThreadButtonNames (opens Collections/Projects)`)
		}
	}
}

func TestErrUIChanged(t *testing.T) {
	err := &perplexity.ErrUIChanged{Op: "mode", Msg: "missing"}
	if err.Error() == "" {
		t.Fatal("empty error")
	}
}

func TestDeepResearchMenuPattern(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"Deep Research", true},
		{"Deep research", true},
		{"deep research", true},
		{"Model Council", false},
		{"Plan mode", false},
	}
	for _, tc := range cases {
		got := perplexity.DeepResearchMenuPattern.MatchString(tc.s)
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.s, got, tc.want)
		}
	}
}

func TestModalityUseButtonPattern(t *testing.T) {
	if !perplexity.ModalityUseButtonPattern.MatchString("Use") {
		t.Fatal("Use should match")
	}
	if perplexity.ModalityUseButtonPattern.MatchString("Reuse") {
		t.Fatal("Reuse should not match ^use$")
	}
}

func TestShouldWarmup(t *testing.T) {
	const warm = "https://www.google.com"
	const base = "https://www.perplexity.ai"
	cases := []struct {
		cur  string
		want bool
	}{
		{"", true},
		{"about:blank", true},
		{"chrome://newtab/", true},
		{"https://www.google.com/", false},
		{"https://www.google.com.au/search?q=x", false},
		{"https://www.perplexity.ai/", false},
		{"https://www.perplexity.ai/search/abc", false},
		{"https://example.com/", true},
	}
	for _, tc := range cases {
		got := perplexity.ShouldWarmup(tc.cur, warm, base)
		if got != tc.want {
			t.Fatalf("cur=%q: got %v want %v", tc.cur, got, tc.want)
		}
	}
	if perplexity.ShouldWarmup("about:blank", "", base) {
		t.Fatal("empty warmup should disable")
	}
}

func containsPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
