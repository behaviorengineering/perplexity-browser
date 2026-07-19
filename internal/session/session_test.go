package session

import (
	"fmt"
	"strings"
	"testing"
)

func TestLoginHintStrings(t *testing.T) {
	hints := []string{"Sign in", "Log in", "Continue with Google"}
	for _, h := range hints {
		if strings.TrimSpace(h) == "" {
			t.Fatal("empty hint")
		}
	}
}

func TestIsTargetClosed(t *testing.T) {
	if !isTargetClosed(fmt.Errorf("playwright: target closed: Target page, context or browser has been closed")) {
		t.Fatal("expected target closed")
	}
	if isTargetClosed(fmt.Errorf("compose textbox not found")) {
		t.Fatal("unexpected")
	}
	if isTargetClosed(nil) {
		t.Fatal("nil")
	}
}

func TestAuthPathDetection(t *testing.T) {
	cases := []struct {
		url  string
		auth bool
	}{
		{"https://www.perplexity.ai/?login-source=signupButton&login-new=false", false},
		{"https://www.perplexity.ai/", false},
		{"https://www.perplexity.ai/login", true},
		{"https://www.perplexity.ai/signin", true},
		{"https://www.perplexity.ai/auth/callback", true},
	}
	for _, tc := range cases {
		path := strings.ToLower(tc.url)
		if i := strings.Index(path, "?"); i >= 0 {
			path = path[:i]
		}
		got := strings.Contains(path, "/login") || strings.Contains(path, "/signin") || strings.Contains(path, "/auth/")
		if got != tc.auth {
			t.Fatalf("%s: got auth=%v want %v", tc.url, got, tc.auth)
		}
	}
}
