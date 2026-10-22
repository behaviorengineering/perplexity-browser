package session

import (
	"context"
	"sync"
	"testing"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/config"
	"github.com/behaviorengineering/perplexity-browser/internal/pagewatch"
)

func TestAttachWatch_nilPageReturnsNil(t *testing.T) {
	m := New(config.Config{}, nil)
	if w := m.attachWatch(nil); w != nil {
		t.Fatal("expected nil watch")
	}
}

type sessFake struct {
	mu       sync.Mutex
	handlers map[string][]func(map[string]any)
}

func (s *sessFake) Send(string, map[string]any) (any, error) { return map[string]any{}, nil }
func (s *sessFake) On(event string, handler func(map[string]any)) {
	s.mu.Lock()
	s.handlers[event] = append(s.handlers[event], handler)
	s.mu.Unlock()
}
func (s *sessFake) Detach() error { return nil }

func (s *sessFake) fire(event string, params map[string]any) {
	s.mu.Lock()
	h := append([]func(map[string]any){}, s.handlers[event]...)
	s.mu.Unlock()
	for _, fn := range h {
		fn(params)
	}
}

type pageStub struct {
	url string
}

func (p *pageStub) URL() string                        { return p.url }
func (p *pageStub) On(string, interface{})             {}
func (p *pageStub) Context() playwright.BrowserContext { return nil }

func TestFinishWatch_copiesResetDetected(t *testing.T) {
	m := New(config.Config{}, nil)
	sf := &sessFake{handlers: make(map[string][]func(map[string]any))}
	page := &pageStub{url: "https://www.perplexity.ai/search/new"}
	w, err := pagewatch.Create(pagewatch.Config{
		Page: page,
		NewSession: func(pagewatch.PageHook) (pagewatch.SessionPort, error) {
			return sf, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	sf.fire("Page.frameRequestedNavigation", map[string]any{
		"reason": "reload",
		"url":    "https://www.perplexity.ai/",
	})
	pw := m.finishWatch(w)
	if pw == nil || !pw.ResetDetected {
		t.Fatalf("expected reset_detected, got %+v", pw)
	}
}
