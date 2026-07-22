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

// Research starts a new thread, sets mode, submits prompt, waits, extracts answer.
// Caller must already hold TryBegin / End around this call.
func (m *Manager) Research(ctx context.Context, prompt, mode string, timeoutMS int) result.Turn {
	start := time.Now()
	out := result.Turn{Base: result.Base{Busy: true}}

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
		return *status
	}

	if err := perplexity.EnsureCompose(page); err != nil {
		return m.mapFlowErr(out, err, mode, start, page)
	}
	if err := perplexity.SetMode(page, mode); err != nil {
		return m.mapFlowErr(out, err, mode, start, page)
	}
	if err := perplexity.SubmitPrompt(page, prompt); err != nil {
		return m.mapFlowErr(out, err, mode, start, page)
	}

	threadID := newThreadID()
	m.mu.Lock()
	m.threadID = threadID
	m.mode = mode
	runCtx := m.runCtx
	m.mu.Unlock()

	waitCtx, stopWait := mergeCancel(ctx, runCtx)
	defer stopWait()
	if err := perplexity.WaitComplete(waitCtx, page, m.cfg.PollMS, timeoutMS); err != nil {
		return m.finishTurn(out, page, threadID, mode, start, err)
	}
	return m.finishTurn(out, page, threadID, mode, start, nil)
}

// Continue sends a follow-up on the active thread and waits for the next answer.
// Caller must already hold TryBegin / End.
func (m *Manager) Continue(ctx context.Context, message, threadID string, timeoutMS int) result.Turn {
	start := time.Now()
	out := result.Turn{Base: result.Base{Busy: true}}

	message = strings.TrimSpace(message)
	if message == "" {
		out.Status = result.StatusError
		out.Message = "message is required"
		out.Busy = false
		return out
	}

	m.mu.Lock()
	page := m.page
	active := m.threadID
	mode := m.mode
	runCtx := m.runCtx
	m.mu.Unlock()

	if page == nil {
		out.Status = result.StatusError
		out.Message = "browser not open; call perplexity_research first"
		out.Busy = false
		return out
	}
	if active == "" {
		out.Status = result.StatusError
		out.Message = "no active thread; call perplexity_research first"
		out.Busy = false
		return out
	}
	if threadID != "" && threadID != active {
		out.Status = result.StatusError
		out.Message = fmt.Sprintf("thread_id mismatch: active %s, requested %s", active, threadID)
		out.Busy = false
		return out
	}
	if mode == "" {
		mode = perplexity.ModeDeep
	}
	timeoutMS = m.resolveTimeout(mode, timeoutMS)

	if err := perplexity.SubmitPrompt(page, message); err != nil {
		return m.mapFlowErr(out, err, mode, start, page)
	}
	waitCtx, stopWait := mergeCancel(ctx, runCtx)
	defer stopWait()
	if err := perplexity.WaitComplete(waitCtx, page, m.cfg.PollMS, timeoutMS); err != nil {
		return m.finishTurn(out, page, active, mode, start, err)
	}
	return m.finishTurn(out, page, active, mode, start, nil)
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

func (m *Manager) finishTurn(out result.Turn, page playwright.Page, threadID, mode string, start time.Time, waitErr error) result.Turn {
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
	if waitErr == nil {
		out.Status = result.StatusOK
		out.Message = "ok"
		return out
	}
	if errors.Is(waitErr, context.Canceled) {
		out.Status = result.StatusCancelled
		out.Message = "cancelled"
		return out
	}
	if errors.Is(waitErr, context.DeadlineExceeded) {
		out.Status = result.StatusTimeout
		out.Message = "timed out; partial answer returned if any"
		return out
	}
	return m.mapFlowErr(out, waitErr, mode, start, page)
}

func (m *Manager) mapFlowErr(out result.Turn, err error, mode string, start time.Time, page playwright.Page) result.Turn {
	out.Mode = mode
	out.ElapsedMS = time.Since(start).Milliseconds()
	out.Busy = false
	if page != nil {
		out.URL = page.URL()
	}
	m.mu.Lock()
	out.ThreadID = m.threadID
	m.mu.Unlock()
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
