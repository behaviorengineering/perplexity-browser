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
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PERPLEXITY_BROWSER_HEADLESS", "1")
	t.Setenv("PERPLEXITY_BROWSER_DEFAULT_TIMEOUT_MS", "60000")
	t.Setenv("PERPLEXITY_BROWSER_BASE_URL", "https://example.test")

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
}
