package perplexity_test

import (
	"strings"
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
)

func TestIsFollowUpChrome(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"Ask a follow-up\nSearch\nComputer\nModel", true},
		{"Ask a follow-up Search Computer Model", true},
		{"ping merriam-webster", false},
		{"Search the logs for errors.", false},
	}
	for _, tc := range cases {
		if got := perplexity.IsFollowUpChrome(tc.s); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.s, got, tc.want)
		}
	}
}

func TestPickAnswerTextPrefersProseOverDock(t *testing.T) {
	t.Parallel()
	got := perplexity.PickAnswerText(
		"Ask a follow-up\nSearch\nComputer\nModel",
		"ping merriam-webster",
	)
	if got != "ping merriam-webster" {
		t.Fatalf("got %q", got)
	}
}

func TestPickAnswerTextAllChromeIsEmpty(t *testing.T) {
	t.Parallel()
	got := perplexity.PickAnswerText("Ask a follow-up Search Computer Model", "")
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestAnswerTextSelectorsRejectClassSubstring(t *testing.T) {
	t.Parallel()
	for _, sel := range perplexity.AnswerTextSelectors {
		if strings.Contains(sel, "class*=") && strings.Contains(sel, "answer") {
			t.Fatalf("greedy class substring %q matches the follow-up dock", sel)
		}
	}
}
