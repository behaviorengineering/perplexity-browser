package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// threadState is persisted per logical session under ~/.perplexity-browser-mcp/sessions/<id>.json.
type threadState struct {
	SessionID string    `json:"session_id"`
	ThreadID  string    `json:"thread_id"`
	URL       string    `json:"url,omitempty"`
	Mode      string    `json:"mode,omitempty"`
	TitleHint string    `json:"title_hint,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *Manager) persistPath(sessionID string) string {
	sid := sanitizeSessionID(sessionID)
	return filepath.Join(m.sessionsDir(), sid+".json")
}

func (m *Manager) saveThreadState(sessionID, threadID, mode, pageURL, titleHint string) {
	sessionID = sanitizeSessionID(sessionID)
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	st := threadState{
		SessionID: sessionID,
		ThreadID:  threadID,
		URL:       strings.TrimSpace(pageURL),
		Mode:      strings.TrimSpace(mode),
		TitleHint: strings.TrimSpace(titleHint),
		UpdatedAt: time.Now().UTC(),
	}
	dir := m.sessionsDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		m.log.Warn("persist thread state: mkdir", "err", err, "session_id", sessionID)
		return
	}
	path := m.persistPath(sessionID)
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		m.log.Warn("persist thread state: marshal", "err", err, "session_id", sessionID)
		return
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		m.log.Warn("persist thread state: write", "err", err, "path", path, "session_id", sessionID)
	}
}

func (m *Manager) loadThreadState(sessionID string) (threadState, bool) {
	sessionID = sanitizeSessionID(sessionID)
	path := m.persistPath(sessionID)
	b, err := os.ReadFile(path)
	if err != nil {
		if sessionID == defaultSessionID {
			if legacy, ok := m.loadLegacyStateFile(); ok {
				return legacy, true
			}
		}
		return threadState{}, false
	}
	var st threadState
	if err := json.Unmarshal(b, &st); err != nil {
		m.log.Warn("load thread state: unmarshal", "err", err, "path", path, "session_id", sessionID)
		return threadState{}, false
	}
	if strings.TrimSpace(st.ThreadID) == "" {
		return threadState{}, false
	}
	if st.SessionID == "" {
		st.SessionID = sessionID
	}
	return st, true
}

func (m *Manager) loadLegacyStateFile() (threadState, bool) {
	path := m.legacyStatePath()
	b, err := os.ReadFile(path)
	if err != nil {
		return threadState{}, false
	}
	var st threadState
	if err := json.Unmarshal(b, &st); err != nil {
		return threadState{}, false
	}
	if strings.TrimSpace(st.ThreadID) == "" {
		return threadState{}, false
	}
	if st.SessionID == "" {
		st.SessionID = defaultSessionID
	}
	return st, true
}

func (m *Manager) ensureThreadFromState(sessionID string) threadState {
	sessionID = m.resolveSessionID(sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeSessionID == sessionID && m.threadID != "" {
		return threadState{
			SessionID: sessionID,
			ThreadID:  m.threadID,
			Mode:      m.mode,
			URL:       m.pageURLLocked(),
		}
	}
	st, ok := m.loadThreadState(sessionID)
	if !ok {
		return threadState{SessionID: sessionID}
	}
	m.threadID = st.ThreadID
	if st.Mode != "" {
		m.mode = st.Mode
	}
	m.activeSessionID = sessionID
	return st
}

func (m *Manager) activateSession(ctx context.Context, sessionID string) (threadState, error) {
	sessionID = m.resolveSessionID(sessionID)
	st, ok := m.loadThreadState(sessionID)
	if !ok {
		return threadState{SessionID: sessionID}, fmt.Errorf("no thread for session %q; run perplexity_research with this session_id first", sessionID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threadID = st.ThreadID
	if st.Mode != "" {
		m.mode = st.Mode
	}
	m.activeSessionID = sessionID
	if err := m.openThreadURLLocked(ctx, st.URL); err != nil {
		return st, err
	}
	return st, nil
}

func (m *Manager) openPersistedThreadURL(ctx context.Context, sessionID string) error {
	sessionID = m.resolveSessionID(sessionID)
	st := m.ensureThreadFromState(sessionID)
	if strings.TrimSpace(st.URL) == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.openThreadURLLocked(ctx, st.URL)
}

func (m *Manager) openThreadURLLocked(ctx context.Context, url string) error {
	_ = ctx
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	if err := m.ensureBrowserLocked(ctx); err != nil {
		return err
	}
	if m.page == nil {
		return fmt.Errorf("no page")
	}
	cur := strings.TrimSpace(m.page.URL())
	if cur == url || strings.HasPrefix(cur, url) {
		return nil
	}
	if _, err := m.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60_000),
	}); err != nil {
		return fmt.Errorf("goto thread url: %w", err)
	}
	time.Sleep(1200 * time.Millisecond)
	return nil
}
