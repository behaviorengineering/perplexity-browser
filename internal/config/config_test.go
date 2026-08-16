package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PERPLEXITY_BROWSER_USER_DATA_DIR", "")
	t.Setenv("PERPLEXITY_BROWSER_EXPORT_DIR", "")
	t.Setenv("PERPLEXITY_BROWSER_HEADLESS", "")
	t.Setenv("PERPLEXITY_BROWSER_BASE_URL", "")
	t.Setenv("PERPLEXITY_BROWSER_WARMUP_URL", "")

	cfg := config.Load()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	wantProfile := filepath.Join(home, ".perplexity-browser-mcp", "profile")
	if cfg.UserDataDir != wantProfile {
		t.Fatalf("UserDataDir = %q, want %q", cfg.UserDataDir, wantProfile)
	}
	if cfg.Headless {
		t.Fatal("expected headed by default")
	}
	if cfg.DefaultTimeoutMS != 900_000 {
		t.Fatalf("DefaultTimeoutMS = %d", cfg.DefaultTimeoutMS)
	}
	if cfg.BaseURL != "https://www.perplexity.ai" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.WarmupURL != "https://www.google.com" {
		t.Fatalf("WarmupURL = %q", cfg.WarmupURL)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PERPLEXITY_BROWSER_HEADLESS", "1")
	t.Setenv("PERPLEXITY_BROWSER_DEFAULT_TIMEOUT_MS", "60000")
	t.Setenv("PERPLEXITY_BROWSER_BASE_URL", "https://example.test")
	t.Setenv("PERPLEXITY_BROWSER_WARMUP_URL", "https://warmup.test")

	cfg := config.Load()
	if !cfg.Headless {
		t.Fatal("expected headless")
	}
	if cfg.DefaultTimeoutMS != 60_000 {
		t.Fatalf("DefaultTimeoutMS = %d", cfg.DefaultTimeoutMS)
	}
	if cfg.BaseURL != "https://example.test" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.WarmupURL != "https://warmup.test" {
		t.Fatalf("WarmupURL = %q", cfg.WarmupURL)
	}
}

func TestLoadWarmupURLDisabled(t *testing.T) {
	// Explicit empty is not possible via envOr; use a sentinel "-" then document
	// that PERPLEXITY_BROWSER_WARMUP_URL=off disables. Prefer off/none/0.
	t.Setenv("PERPLEXITY_BROWSER_WARMUP_URL", "off")
	cfg := config.Load()
	if cfg.WarmupURL != "" {
		t.Fatalf("WarmupURL should be disabled, got %q", cfg.WarmupURL)
	}
}
