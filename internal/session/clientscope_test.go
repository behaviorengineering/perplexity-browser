package session

import "testing"

func TestScopeFromRootURI(t *testing.T) {
	t.Parallel()
	if got := scopeFromRootURI("file:///Users/me/projects/cr-case-intake"); got != "cr-case-intake" {
		t.Fatalf("got %q", got)
	}
	if got := scopeFromRootURI(""); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := scopeFromRootURI("https://example.com"); got != "" {
		t.Fatalf("got %q", got)
	}
}
