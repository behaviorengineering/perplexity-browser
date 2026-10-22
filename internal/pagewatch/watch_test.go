package pagewatch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
)

type fakeSession struct {
	mu       sync.Mutex
	sent     []string
	handlers map[string][]func(map[string]any)
	sendErr  map[string]error
}

func newFakeSession() *fakeSession {
	return &fakeSession{
		handlers: make(map[string][]func(map[string]any)),
		sendErr:  make(map[string]error),
	}
}

func (f *fakeSession) Send(method string, params map[string]any) (any, error) {
	f.mu.Lock()
	f.sent = append(f.sent, method)
	err := f.sendErr[method]
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (f *fakeSession) On(event string, handler func(params map[string]any)) {
	f.mu.Lock()
	f.handlers[event] = append(f.handlers[event], handler)
	f.mu.Unlock()
}

func (f *fakeSession) Detach() error { return nil }

func (f *fakeSession) emit(event string, params map[string]any) {
	f.mu.Lock()
	handlers := append([]func(map[string]any){}, f.handlers[event]...)
	f.mu.Unlock()
	for _, h := range handlers {
		h(params)
	}
}

type stubPage struct {
	url      string
	handlers map[string][]func(interface{})
}

func newStubPage(url string) *stubPage {
	return &stubPage{url: url, handlers: make(map[string][]func(interface{}))}
}

func (s *stubPage) URL() string { return s.url }

func (s *stubPage) On(event string, handler interface{}) {
	fn, ok := handler.(func(playwright.Frame))
	if ok {
		s.handlers[event] = append(s.handlers[event], func(v interface{}) { fn(v.(playwright.Frame)) })
		return
	}
	if h, ok := handler.(func(error)); ok {
		s.handlers[event] = append(s.handlers[event], func(v interface{}) {
			if v == nil {
				h(nil)
			} else if e, ok := v.(error); ok {
				h(e)
			}
		})
		return
	}
	if h, ok := handler.(func()); ok {
		s.handlers[event] = append(s.handlers[event], func(interface{}) { h() })
		return
	}
	if h, ok := handler.(func(playwright.Request)); ok {
		s.handlers[event] = append(s.handlers[event], func(v interface{}) {
			if r, ok := v.(playwright.Request); ok {
				h(r)
			}
		})
	}
}

func (s *stubPage) Context() playwright.BrowserContext { return nil }

func TestWatch_cdpReloadEventInSnapshot(t *testing.T) {
	fs := newFakeSession()
	page := newStubPage("https://www.perplexity.ai/search/new")
	w, err := Create(Config{
		Page: page,
		Now:  func() time.Time { return time.Unix(0, 0) },
		NewSession: func(PageHook) (sessionAPI, error) {
			return fs, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	fs.emit("Page.frameRequestedNavigation", map[string]any{
		"reason": "reload",
		"url":    "https://www.perplexity.ai/",
	})
	snap := w.Snapshot()
	if len(snap.Events) == 0 || snap.Events[0].Kind != "reload" {
		t.Fatalf("snapshot: %+v", snap)
	}
	_ = w.Stop()
}

func TestWatch_stopIdempotent(t *testing.T) {
	fs := newFakeSession()
	page := newStubPage("https://www.perplexity.ai/")
	w, err := Create(Config{
		Page: page,
		NewSession: func(PageHook) (sessionAPI, error) {
			return fs, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Start(context.Background())
	if err := w.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := w.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestWatch_enableFailUsesFallbackBackend(t *testing.T) {
	fs := newFakeSession()
	fs.sendErr["Page.enable"] = errors.New("cdp down")
	page := newStubPage("https://www.perplexity.ai/search/new")
	w, err := Create(Config{
		Page: page,
		NewSession: func(PageHook) (sessionAPI, error) {
			return fs, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := w.Snapshot()
	if snap.Backend != "playwright" {
		t.Fatalf("backend=%q", snap.Backend)
	}
	_ = w.Stop()
}

func TestWatch_handlersIgnoreAfterStop(t *testing.T) {
	fs := newFakeSession()
	page := newStubPage("https://www.perplexity.ai/search/new")
	w, err := Create(Config{
		Page: page,
		NewSession: func(PageHook) (sessionAPI, error) {
			return fs, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Start(context.Background())
	_ = w.Stop()
	fs.emit("Page.frameRequestedNavigation", map[string]any{
		"reason": "reload",
		"url":    "https://www.perplexity.ai/",
	})
	snap := w.Snapshot()
	if len(snap.Events) != 0 {
		t.Fatalf("expected no events after stop, got %+v", snap.Events)
	}
}
