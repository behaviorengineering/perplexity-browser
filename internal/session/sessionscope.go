package session

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	defaultSessionID = "default"
	maxSessionIDLen  = 64
)

var sessionIDSafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// sanitizeSessionID maps a caller session id to a safe filename segment.
func sanitizeSessionID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return defaultSessionID
	}
	id = sessionIDSafe.ReplaceAllString(id, "_")
	if len(id) > maxSessionIDLen {
		id = id[:maxSessionIDLen]
	}
	id = strings.Trim(id, "._-")
	if id == "" {
		return defaultSessionID
	}
	return id
}

func (m *Manager) resolveSessionID(override string) string {
	if s := strings.TrimSpace(override); s != "" {
		return sanitizeSessionID(s)
	}
	if m.defaultSessionID != "" {
		return m.defaultSessionID
	}
	return defaultSessionID
}

func (m *Manager) sessionsDir() string {
	if d := strings.TrimSpace(m.cfg.SessionsDir); d != "" {
		return d
	}
	return filepath.Join(filepath.Dir(m.cfg.UserDataDir), "sessions")
}

func (m *Manager) legacyStatePath() string {
	if p := strings.TrimSpace(m.cfg.StatePath); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(m.cfg.UserDataDir), "state.json")
}

func (m *Manager) setActiveSession(sessionID string) {
	m.mu.Lock()
	m.activeSessionID = sessionID
	m.mu.Unlock()
}

func (m *Manager) activeSession() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeSessionID
}

// SanitizeSessionIDForTest exposes session id sanitization for external tests.
func SanitizeSessionIDForTest(id string) string { return sanitizeSessionID(id) }
