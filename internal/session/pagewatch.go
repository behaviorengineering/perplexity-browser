package session

import (
	"context"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/pagewatch"
	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

func (m *Manager) attachWatch(page playwright.Page) *pagewatch.Watch {
	if page == nil {
		return nil
	}
	w, err := pagewatch.Create(pagewatch.Config{
		Page: page,
		Log:  m.log,
	})
	if err != nil {
		m.log.Warn("pagewatch start failed", "err", err)
		return nil
	}
	if err := w.Start(context.Background()); err != nil {
		m.log.Warn("pagewatch start failed", "err", err)
		return nil
	}
	return w
}

func (m *Manager) finishWatch(w *pagewatch.Watch) *result.PageWatch {
	if w == nil {
		return nil
	}
	snap := w.Snapshot()
	_ = w.Stop()
	out := snap
	m.mu.Lock()
	m.lastWatch = &out
	m.mu.Unlock()
	return &out
}

func (m *Manager) copyLastWatch() *result.PageWatch {
	m.mu.Lock()
	lw := m.lastWatch
	m.mu.Unlock()
	if lw == nil {
		return nil
	}
	cp := *lw
	return &cp
}
