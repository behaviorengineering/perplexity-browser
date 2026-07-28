package session_test

import (
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/session"
)

func TestSanitizeSessionIDExported(t *testing.T) {
	t.Parallel()
	if got := session.SanitizeSessionIDForTest("My Project"); got != "My_Project" {
		t.Fatalf("got %q", got)
	}
}
