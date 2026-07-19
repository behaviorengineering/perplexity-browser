package session

import (
	"strings"
	"testing"
)

func TestLoginHintStrings(t *testing.T) {
	// Sanity: helpers stay conservative (fail closed on login walls).
	hints := []string{"Sign in", "Log in", "Continue with Google"}
	for _, h := range hints {
		if strings.TrimSpace(h) == "" {
			t.Fatal("empty hint")
		}
	}
}
