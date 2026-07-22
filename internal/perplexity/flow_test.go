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

func TestErrUIChanged(t *testing.T) {
	err := &perplexity.ErrUIChanged{Op: "mode", Msg: "missing"}
	if err.Error() == "" {
		t.Fatal("empty error")
	}
}

func containsPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
