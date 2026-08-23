package session

import (
	"path/filepath"
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/config"
)

func TestSaveAndLoadThreadStatePerSession(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.Config{
		UserDataDir: filepath.Join(dir, "profile"),
		SessionsDir: filepath.Join(dir, "sessions"),
		SessionID:   "default",
	}
	m := New(cfg, nil)
	m.saveThreadState("cr-case", "thr_a", "deep", "https://www.perplexity.ai/search/a", "pack A")
	m.saveThreadState("other", "thr_b", "search", "https://www.perplexity.ai/search/b", "pack B")

	a, ok := m.loadThreadState("cr-case")
	if !ok || a.ThreadID != "thr_a" || a.URL == "" {
		t.Fatalf("cr-case state %+v ok=%v", a, ok)
	}
	b, ok := m.loadThreadState("other")
	if !ok || b.ThreadID != "thr_b" {
		t.Fatalf("other state %+v ok=%v", b, ok)
	}
}

func TestEnsureThreadFromStateHydratesMemory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.Config{
		UserDataDir: filepath.Join(dir, "profile"),
		SessionsDir: filepath.Join(dir, "sessions"),
		SessionID:   "cr-case",
	}
	m1 := New(cfg, nil)
	m1.saveThreadState("cr-case", "thr_hydrate", "search", "https://www.perplexity.ai/search/hydrate", "")

	m2 := New(cfg, nil)
	st := m2.ensureThreadFromState("cr-case")
	if st.ThreadID != "thr_hydrate" || m2.threadID != "thr_hydrate" {
		t.Fatalf("got %+v threadID=%q", st, m2.threadID)
	}
}

func TestSanitizeSessionID(t *testing.T) {
	t.Parallel()
	if sanitizeSessionID("cr-case-intake") != "cr-case-intake" {
		t.Fatal("expected safe id")
	}
	if sanitizeSessionID("bad id!") != "bad_id" {
		t.Fatalf("got %q", sanitizeSessionID("bad id!"))
	}
}
