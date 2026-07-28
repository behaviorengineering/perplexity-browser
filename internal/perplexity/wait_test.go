package perplexity

import (
	"testing"
	"time"
)

func TestWaitTracker(t *testing.T) {
	t.Parallel()
	var tracker waitTracker

	if tracker.tick(false, 0, 2) {
		t.Fatal("empty answer should not complete")
	}
	if tracker.tick(true, 10, 2) {
		t.Fatal("generating should not complete")
	}
	if tracker.tick(false, 50, 2) {
		t.Fatal("first stable poll should not complete")
	}
	if tracker.tick(false, 50, 2) {
		t.Fatal("second stable poll should not complete yet")
	}
	if !tracker.tick(false, 50, 2) {
		t.Fatal("third stable poll should complete")
	}
}

func TestPollInterval(t *testing.T) {
	t.Parallel()
	fast := pollInterval(true, 800, 350)
	slow := pollInterval(false, 800, 350)
	if fast >= slow {
		t.Fatalf("fast=%v slow=%v", fast, slow)
	}
	if fast < 200*time.Millisecond {
		t.Fatalf("fast too small: %v", fast)
	}
}

func TestWaitSettingsNormalize(t *testing.T) {
	t.Parallel()
	s := (WaitSettings{}).normalize()
	if s.IdlePollMS != 800 || s.FastPollMS != 350 || s.StablePolls != 2 {
		t.Fatalf("defaults: %+v", s)
	}
}
