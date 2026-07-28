// Package config loads env-based settings for the Perplexity browser MCP.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds runtime paths and timeouts.
type Config struct {
	UserDataDir      string
	ExportDir        string
	SessionID        string
	SessionsDir      string
	StatePath        string // legacy single-file state (migrated into sessions/default.json)
	BaseURL          string
	Headless         bool
	DefaultTimeoutMS int
	SearchTimeoutMS  int
	PollMS           int
	PollFastMS       int
	StablePolls      int
	AnswerMaxChars   int
	Channel          string
	CDPURL           string
	CDPAutoLaunch    bool
	ChromeApp        string
	LogPrompts       bool
}

// Load reads environment variables with defaults from the technical appendix.
func Load() Config {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".perplexity-browser-mcp")

	return Config{
		UserDataDir:      envOr("PERPLEXITY_BROWSER_USER_DATA_DIR", filepath.Join(root, "profile")),
		ExportDir:        envOr("PERPLEXITY_BROWSER_EXPORT_DIR", filepath.Join(root, "exports")),
		StatePath:        envOr("PERPLEXITY_BROWSER_STATE_PATH", filepath.Join(root, "state.json")),
		SessionID:        sanitizeSessionIDEnv(envOr("PERPLEXITY_BROWSER_SESSION_ID", "default")),
		SessionsDir:      envOr("PERPLEXITY_BROWSER_SESSIONS_DIR", filepath.Join(root, "sessions")),
		BaseURL:          envOr("PERPLEXITY_BROWSER_BASE_URL", "https://www.perplexity.ai"),
		Headless:         envBool("PERPLEXITY_BROWSER_HEADLESS", false),
		DefaultTimeoutMS: envInt("PERPLEXITY_BROWSER_DEFAULT_TIMEOUT_MS", 900_000),
		SearchTimeoutMS:  envInt("PERPLEXITY_BROWSER_SEARCH_TIMEOUT_MS", 180_000),
		PollMS:           envInt("PERPLEXITY_BROWSER_POLL_MS", 800),
		PollFastMS:       envInt("PERPLEXITY_BROWSER_POLL_FAST_MS", 350),
		StablePolls:      envInt("PERPLEXITY_BROWSER_STABLE_POLLS", 2),
		AnswerMaxChars:   envInt("PERPLEXITY_BROWSER_ANSWER_MAX_CHARS", 120_000),
		Channel:          strings.TrimSpace(os.Getenv("PERPLEXITY_BROWSER_CHANNEL")),
		CDPURL:           strings.TrimSpace(os.Getenv("PERPLEXITY_BROWSER_CDP_URL")),
		CDPAutoLaunch:    envBool("PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH", true),
		ChromeApp:        envOr("PERPLEXITY_BROWSER_CHROME_APP", defaultChromeApp()),
		LogPrompts:       envBool("PERPLEXITY_BROWSER_LOG_PROMPTS", false),
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func sanitizeSessionIDEnv(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "default"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
