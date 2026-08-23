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

func TestWaitTrackerNoiseBand(t *testing.T) {
	t.Parallel()
	var tracker waitTracker
	_ = tracker.tick(false, 1000, 2)
	_ = tracker.tick(false, 1000, 2)
	// Related-sources churn under noise band should still complete.
	if !tracker.tick(false, 1080, 2) {
		t.Fatal("small length churn should count as stable and complete")
	}
}

func TestWaitTrackerIdleFailsafe(t *testing.T) {
	t.Parallel()
	var tracker waitTracker
	// Keep twitching just above noise so stable never reaches required, but
	// idle-not-generating failsafe should still finish.
	lens := []int{500, 650, 800, 950, 1100, 1250}
	done := false
	for _, n := range lens {
		if tracker.tick(false, n, 2) {
			done = true
			break
		}
	}
	if !done {
		t.Fatal("idle failsafe should complete after several non-generating polls")
	}
}

func TestLooksLikeStopGenerating(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    string
		want bool
	}{
		{"Stop generating", true},
		{"Stop", true},
		{"Cancel", false},
		{"Account", false},
		{"Stopwatch", false},
		{"Stop research", true},
	}
	for _, tc := range cases {
		if got := looksLikeStopGenerating(tc.s); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.s, got, tc.want)
		}
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
