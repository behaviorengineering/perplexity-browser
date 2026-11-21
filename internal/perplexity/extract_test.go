package perplexity_test

import (
	"strings"
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
)

func TestNormalizeUserPrompt(t *testing.T) {
	t.Parallel()
	const prompt = "I am a solo freelancer / small business selling professional services."
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{prompt, prompt},
		{"09:44\n\n" + prompt + "\n\nRead more", prompt},
		{"9:05\n" + prompt, prompt},
		{prompt + "\nRead more", prompt},
		{"Read more", ""},
		{"Ask about Read more in the docs.", "Ask about Read more in the docs."},
	}
	for _, tc := range cases {
		if got := perplexity.NormalizeUserPrompt(tc.in); got != tc.want {
			t.Fatalf("in %q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

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
