package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

func (m *Manager) stampSession(out *result.Base, sessionID string) {
	if out == nil {
		return
	}
	out.SessionID = sessionID
	out.ActiveSessionID = m.activeSession()
}

// Research starts a new thread, sets mode, submits prompt, waits, extracts answer.
// Caller must already hold TryBegin / End around this call.
func (m *Manager) Research(ctx context.Context, prompt, mode, titleHint, sessionID string, timeoutMS int) result.Turn {
	sessionID = m.resolveSessionID(sessionID)
	m.setActiveSession(sessionID)

	start := time.Now()
	out := result.Turn{Base: result.Base{Busy: true}}
	m.stampSession(&out.Base, sessionID)

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		out.Status = result.StatusError
		out.Message = "prompt is required"
		out.Busy = false
		return out
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = perplexity.ModeDeep
	}
	timeoutMS = m.resolveTimeout(mode, timeoutMS)

	page, status := m.preparePage(ctx)
	if status != nil {
		status.Mode = mode
		status.ElapsedMS = time.Since(start).Milliseconds()
		status.Busy = false
		m.stampSession(&status.Base, sessionID)
		return *status
	}

	if err := perplexity.EnsureCompose(page); err != nil {
		return m.mapFlowErr(out, err, mode, start, page, sessionID)
	}
	if err := perplexity.SetMode(page, mode); err != nil {
		return m.mapFlowErr(out, err, mode, start, page, sessionID)
	}
	if err := perplexity.SubmitPrompt(page, prompt); err != nil {
		return m.mapFlowErr(out, err, mode, start, page, sessionID)
	}

	threadID := newThreadID()
	m.mu.Lock()
	m.threadID = threadID
	m.mode = mode
	m.activeSessionID = sessionID
	runCtx := m.runCtx
	m.mu.Unlock()

	waitCtx, stopWait := mergeCancel(ctx, runCtx)
	defer stopWait()
	if err := perplexity.WaitComplete(waitCtx, page, m.waitSettings(timeoutMS)); err != nil {
		return m.finishTurn(out, page, sessionID, threadID, mode, start, err, titleHint)
	}
	return m.finishTurn(out, page, sessionID, threadID, mode, start, nil, titleHint)
}

// Continue sends a follow-up on the active thread and waits for the next answer.
// Caller must already hold TryBegin / End.
func (m *Manager) Continue(ctx context.Context, message, threadID, sessionID string, timeoutMS int) result.Turn {
	sessionID = m.resolveSessionID(sessionID)
	start := time.Now()
	out := result.Turn{Base: result.Base{Busy: true}}
	m.stampSession(&out.Base, sessionID)

	message = strings.TrimSpace(message)
	if message == "" {
		out.Status = result.StatusError
		out.Message = "message is required"
		out.Busy = false
		return out
	}

	st, err := m.activateSession(ctx, sessionID)
	if err != nil {
		out.Status = result.StatusError
		out.Message = err.Error()
		out.Busy = false
		return out
	}

	m.mu.Lock()
	page := m.page
	active := m.threadID
	mode := m.mode
	runCtx := m.runCtx
	m.mu.Unlock()

	if threadID != "" && active != "" && threadID != active {
		out.Status = result.StatusError
		out.Message = fmt.Sprintf("thread_id mismatch: session %q has %s, requested %s", sessionID, active, threadID)
		out.Busy = false
		return out
	}
	if threadID != "" && st.ThreadID != "" && threadID != st.ThreadID {
		out.Status = result.StatusError
		out.Message = fmt.Sprintf("thread_id does not belong to session %q", sessionID)
		out.Busy = false
		return out
	}

	if page == nil {
		out.Status = result.StatusError
		out.Message = "browser not open; call perplexity_research first"
		out.Busy = false
		return out
	}
	if active == "" {
		out.Status = result.StatusError
		out.Message = fmt.Sprintf("no active thread for session %q; call perplexity_research first", sessionID)
		out.Busy = false
		return out
	}
	if mode == "" {
		mode = perplexity.ModeDeep
	}
	timeoutMS = m.resolveTimeout(mode, timeoutMS)

	if err := perplexity.SubmitPrompt(page, message); err != nil {
		return m.mapFlowErr(out, err, mode, start, page, sessionID)
	}
	waitCtx, stopWait := mergeCancel(ctx, runCtx)
	defer stopWait()
	if err := perplexity.WaitComplete(waitCtx, page, m.waitSettings(timeoutMS)); err != nil {
		return m.finishTurn(out, page, sessionID, active, mode, start, err, "")
	}
	return m.finishTurn(out, page, sessionID, active, mode, start, nil, "")
}

func (m *Manager) preparePage(ctx context.Context) (playwright.Page, *result.Turn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureBrowserLocked(ctx); err != nil {
		return nil, &result.Turn{Base: result.Base{Status: result.StatusError, Message: err.Error()}}
	}
	if err := m.gotoHomeLocked(ctx); err != nil {
		return nil, &result.Turn{Base: result.Base{Status: result.StatusError, Message: err.Error(), URL: m.page.URL()}}
	}
	loggedIn, err := detectLogin(m.page)
	if err != nil {
		return nil, &result.Turn{Base: result.Base{Status: result.StatusError, Message: err.Error(), URL: m.page.URL()}}
	}
	if !loggedIn {
		return nil, &result.Turn{Base: result.Base{
			Status:  result.StatusNeedLogin,
			Message: "sign in in the headed browser window, then call wait_for_login or retry",
			URL:     m.page.URL(),
		}}
	}
	return m.page, nil
}

func (m *Manager) resolveTimeout(mode string, timeoutMS int) int {
	if timeoutMS <= 0 {
		if mode == perplexity.ModeSearch {
			timeoutMS = m.cfg.SearchTimeoutMS
		} else {
			timeoutMS = m.cfg.DefaultTimeoutMS
		}
	}
	if timeoutMS > 30*60*1000 {
		timeoutMS = 30 * 60 * 1000
	}
	return timeoutMS
}

func (m *Manager) waitSettings(timeoutMS int) perplexity.WaitSettings {
	return perplexity.WaitSettings{
		IdlePollMS:  m.cfg.PollMS,
		FastPollMS:  m.cfg.PollFastMS,
		StablePolls: m.cfg.StablePolls,
		TimeoutMS:   timeoutMS,
	}
}

func (m *Manager) finishTurn(out result.Turn, page playwright.Page, sessionID, threadID, mode string, start time.Time, waitErr error, titleHint string) result.Turn {
	ans, _ := perplexity.ExtractAnswerText(page)
	ans, trunc := perplexity.Truncate(ans, m.cfg.AnswerMaxChars)
	out.ThreadID = threadID
	out.Mode = mode
	out.URL = page.URL()
	out.Answer = ans
	out.AnswerTruncated = trunc
	out.Citations = perplexity.ExtractCitations(page)
	out.ElapsedMS = time.Since(start).Milliseconds()
	out.Busy = false
	m.stampSession(&out.Base, sessionID)
	switch {
	case waitErr == nil:
		out.Status = result.StatusOK
		out.Message = "ok"
	case errors.Is(waitErr, context.Canceled):
		out.Status = result.StatusCancelled
		out.Message = "cancelled"
	case errors.Is(waitErr, context.DeadlineExceeded):
		out.Status = result.StatusTimeout
		out.Message = "timed out; partial answer returned if any"
	default:
		return m.mapFlowErr(out, waitErr, mode, start, page, sessionID)
	}
	m.persistTurn(page, sessionID, threadID, mode, titleHint, waitErr)
	return out
}

func (m *Manager) persistTurn(page playwright.Page, sessionID, threadID, mode, titleHint string, waitErr error) {
	if page == nil || threadID == "" {
		return
	}
	hint := ""
	if waitErr == nil {
		_ = perplexity.ApplyTitleHint(page, titleHint)
		hint = titleHint
	}
	m.saveThreadState(sessionID, threadID, mode, page.URL(), hint)
}

func (m *Manager) mapFlowErr(out result.Turn, err error, mode string, start time.Time, page playwright.Page, sessionID string) result.Turn {
	out.Mode = mode
	out.ElapsedMS = time.Since(start).Milliseconds()
	out.Busy = false
	if page != nil {
		out.URL = page.URL()
	}
	m.mu.Lock()
	out.ThreadID = m.threadID
	m.mu.Unlock()
	m.stampSession(&out.Base, sessionID)
	var ui *perplexity.ErrUIChanged
	if errors.As(err, &ui) {
		out.Status = result.StatusUIChanged
		out.Message = ui.Error()
		return out
	}
	out.Status = result.StatusError
	out.Message = err.Error()
	return out
}

func mergeCancel(parent, run context.Context) (context.Context, context.CancelFunc) {
	if run == nil {
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel
	}
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(run, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

func newThreadID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "thr_" + hex.EncodeToString(b[:])
}
