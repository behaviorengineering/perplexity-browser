package pagewatch

import (
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

func TestIsComposeResetURL(t *testing.T) {
	cases := []struct {
		u    string
		want bool
	}{
		{"", true},
		{"about:blank", true},
		{"chrome-error://chromewebdata/", true},
		{"https://www.perplexity.ai/", true},
		{"https://www.perplexity.ai/search/new", true},
		{"https://www.perplexity.ai/search", true},
		{"https://www.perplexity.ai/search/abc123", false},
		{"https://perplexity.ai/search/thread-id/extra", false},
	}
	for _, tc := range cases {
		if got := IsComposeResetURL(tc.u); got != tc.want {
			t.Errorf("IsComposeResetURL(%q) = %v, want %v", tc.u, got, tc.want)
		}
	}
}

func TestIsInThreadURL(t *testing.T) {
	cases := []struct {
		u    string
		want bool
	}{
		{"https://www.perplexity.ai/search/new", false},
		{"https://www.perplexity.ai/search", false},
		{"https://www.perplexity.ai/", false},
		{"https://www.perplexity.ai/search/abc", true},
		{"https://www.perplexity.ai/search/abc/foo", true},
	}
	for _, tc := range cases {
		if got := IsInThreadURL(tc.u); got != tc.want {
			t.Errorf("IsInThreadURL(%q) = %v, want %v", tc.u, got, tc.want)
		}
	}
}

func TestClassifyCDP_reloadReason(t *testing.T) {
	ev, ok := ClassifyCDP("Page.frameRequestedNavigation", map[string]any{
		"reason": "reload",
		"url":    "https://www.perplexity.ai/",
	}, "https://www.perplexity.ai/search/x", 1)
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Kind != "reload" || ev.Reason != "reload" {
		t.Fatalf("got %+v", ev)
	}
}

func TestClassifyCDP_scriptInitiated(t *testing.T) {
	ev, ok := ClassifyCDP("Page.frameRequestedNavigation", map[string]any{
		"reason": "scriptInitiated",
		"url":    "https://www.perplexity.ai/",
	}, "https://www.perplexity.ai/search/x", 1)
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Kind != "script" {
		t.Fatalf("got %+v", ev)
	}
}

func TestClassifyCDP_childFrameSkipped(t *testing.T) {
	_, ok := ClassifyCDP("Page.frameNavigated", map[string]any{
		"frame": map[string]any{
			"url":      "https://example.com/iframe",
			"parentId": "parent-frame-id",
		},
	}, "https://www.perplexity.ai/search/x", 1)
	if ok {
		t.Fatal("expected child frame to be skipped")
	}
}

func TestResetDetected_homeAfterSubmit(t *testing.T) {
	started := "https://www.perplexity.ai/search/new"
	ended := "https://www.perplexity.ai/"
	events := []result.PageEvent{
		{Kind: "reload", URL: "https://www.perplexity.ai/", Reason: "reload"},
	}
	if !ResetDetected(started, ended, events) {
		t.Fatal("expected reset")
	}
}

func TestResetDetected_searchIdIsNotReset(t *testing.T) {
	started := "https://www.perplexity.ai/search/new"
	ended := "https://www.perplexity.ai/search/thread-abc"
	events := []result.PageEvent{
		{Kind: "navigation", URL: ended},
	}
	if ResetDetected(started, ended, events) {
		t.Fatal("expected no reset for thread navigation")
	}
}

func TestErrorDetected_pageerrorWithoutReset(t *testing.T) {
	events := []result.PageEvent{{Kind: "pageerror", Message: "boom"}}
	if !ErrorDetected(events) {
		t.Fatal("expected error_detected")
	}
	started := "https://www.perplexity.ai/search/abc"
	ended := "https://www.perplexity.ai/search/abc"
	if ResetDetected(started, ended, events) {
		t.Fatal("pageerror alone must not set reset")
	}
}
