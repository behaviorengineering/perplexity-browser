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
