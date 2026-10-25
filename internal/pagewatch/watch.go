package pagewatch

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

const (
	defaultMaxEvents = 32
	maxEventsCap     = 128
)

// PageHook is the page surface Watch uses (playwright.Page satisfies this).
type PageHook interface {
	URL() string
	On(event string, handler interface{})
	Context() playwright.BrowserContext
}

// Config wires a one-turn page watcher.
type Config struct {
	Page       PageHook
	Now        func() time.Time
	Log        *slog.Logger
	MaxEvents  int
	NewSession func(PageHook) (SessionPort, error)
}

// Watch records navigation and error events for a single research turn.
type Watch struct {
	cfg        Config
	log        *slog.Logger
	mu         sync.Mutex
	page       PageHook
	startedURL string
	lastURL    string
	events     []result.PageEvent
	truncated  bool
	backend    string
	started    bool
	stopped    bool
	watchID    uint64
	activeID   uint64
	session    sessionAPI
	pending    *pendingNav
	initiator  string
	fallbackOn bool
}

type pendingNav struct {
	ev      result.PageEvent
	at      time.Time
	kindFam string
}

// Create builds a Watch from Config. Nil page returns CodeInvalid.
func Create(cfg Config) (*Watch, error) {
	if cfg.Page == nil {
		return nil, newErr("Create", CodeInvalid, "page is required", nil)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.MaxEvents <= 0 {
		cfg.MaxEvents = defaultMaxEvents
	}
	if cfg.MaxEvents > maxEventsCap {
		cfg.MaxEvents = maxEventsCap
	}
	if cfg.NewSession == nil {
		cfg.NewSession = newPlaywrightSession
	}
	return &Watch{cfg: cfg, log: cfg.Log, page: cfg.Page}, nil
}

// Start enables listeners. Idempotent start returns CodeConflict.
func (w *Watch) Start(ctx context.Context) error {
	if w == nil {
		return newErr("Start", CodeInvalid, "nil watch", nil)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return newErr("Start", CodeConflict, "already started", nil)
	}
	w.watchID++
	w.activeID = w.watchID
	w.started = true
	w.stopped = false
	w.startedURL = w.page.URL()
	w.lastURL = w.startedURL
	id := w.activeID
	w.mu.Unlock()

	sess, err := w.cfg.NewSession(w.page)
	if err != nil {
		w.log.Warn("pagewatch cdp session failed; using playwright fallback", "err", err)
		return w.startPlaywrightFallback(id)
	}
	if err := enableCDP(ctx, sess); err != nil {
		_ = sess.Detach()
		w.log.Warn("pagewatch cdp enable failed; using playwright fallback", "err", err)
		return w.startPlaywrightFallback(id)
	}
	w.mu.Lock()
	w.session = sess
	w.backend = "cdp"
	w.mu.Unlock()

	w.registerCDPHandlers(sess, id)
	return nil
}

func enableCDP(ctx context.Context, sess sessionAPI) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, method := range []string{"Page.enable", "Network.enable", "Runtime.enable"} {
		if _, err := sess.Send(method, nil); err != nil {
			return newErr("Start", CodeUnavailable, method, err)
		}
	}
	return nil
}

func (w *Watch) registerCDPHandlers(sess sessionAPI, id uint64) {
	events := []string{
		"Page.frameRequestedNavigation",
		"Page.frameNavigated",
		"Runtime.exceptionThrown",
		"Network.requestWillBeSent",
		"Network.loadingFailed",
	}
	for _, name := range events {
		sess.On(name, func(params map[string]any) {
			w.safeHandle(id, func() {
				w.handleCDP(name, params)
			})
		})
	}
}

func (w *Watch) handleCDP(eventName string, params map[string]any) {
	w.mu.Lock()
	fromURL := w.lastURL
	if w.page != nil {
		fromURL = w.page.URL()
	}
	atMS := w.cfg.Now().UnixMilli()
	w.mu.Unlock()

	if eventName == "Network.requestWillBeSent" {
		w.noteInitiator(params)
		return
	}

	ev, ok := ClassifyCDP(eventName, params, fromURL, atMS)
	if !ok {
		return
	}
	if eventName == "Page.frameNavigated" {
		w.applyNavigatedCoalesce(ev)
		return
	}
	if eventName == "Page.frameRequestedNavigation" {
		w.mu.Lock()
		w.pending = &pendingNav{
			ev:      ev,
			at:      w.cfg.Now(),
			kindFam: KindFamily(ev.Kind),
		}
		w.mu.Unlock()
		w.appendEvent(ev)
		return
	}
	w.appendEvent(ev)
}

func (w *Watch) noteInitiator(params map[string]any) {
	initType := ""
	if init, ok := mapParam(params, "initiator"); ok && init != nil {
		initType, _ = stringParam(init, "type")
	}
	reqType, _ := stringParam(params, "type")
	if reqType != "" && !strings.EqualFold(reqType, "Document") {
		return
	}
	if req, ok := mapParam(params, "request"); ok && req != nil {
		if rt, ok := stringParam(req, "resourceType"); ok && rt != "" && !strings.EqualFold(rt, "Document") {
			return
		}
	}
	w.mu.Lock()
	if initType != "" {
		w.initiator = initType
	}
	w.mu.Unlock()
}

func (w *Watch) applyNavigatedCoalesce(ev result.PageEvent) {
	w.mu.Lock()
	if w.initiator != "" {
		ev.Initiator = w.initiator
		w.initiator = ""
	}
	pend := w.pending
	w.mu.Unlock()

	if pend != nil && urlsMatch(pend.ev.URL, ev.URL) && w.cfg.Now().Sub(pend.at) <= CoalesceWindow {
		ev.Reason = pend.ev.Reason
		if pend.ev.Kind == "reload" || pend.ev.Kind == "script" {
			ev.Kind = pend.ev.Kind
		}
		w.mu.Lock()
		w.pending = nil
		w.mu.Unlock()
		w.replaceLastCoalesced(ev)
		return
	}
	w.appendEvent(ev)
}

func (w *Watch) replaceLastCoalesced(ev result.PageEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.events) == 0 {
		w.pushEventLocked(ev)
		return
	}
	last := &w.events[len(w.events)-1]
	*last = ev
	if ev.URL != "" {
		w.lastURL = ev.URL
	}
}

func (w *Watch) appendEvent(ev result.PageEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.shouldCoalesceLocked(ev) {
		return
	}
	w.pushEventLocked(ev)
}

func (w *Watch) shouldCoalesceLocked(ev result.PageEvent) bool {
	if len(w.events) == 0 {
		return false
	}
	now := w.cfg.Now()
	fam := KindFamily(ev.Kind)
	for i := len(w.events) - 1; i >= 0; i-- {
		prev := w.events[i]
		if now.Sub(time.UnixMilli(prev.AtMS)) > CoalesceWindow {
			break
		}
		if KindFamily(prev.Kind) == fam && urlsMatch(prev.URL, ev.URL) {
			return true
		}
	}
	return false
}

func (w *Watch) pushEventLocked(ev result.PageEvent) {
	if len(w.events) >= w.cfg.MaxEvents {
		w.events = w.events[1:]
		w.truncated = true
	}
	w.events = append(w.events, ev)
	if ev.URL != "" {
		w.lastURL = ev.URL
	}
}

func (w *Watch) startPlaywrightFallback(id uint64) error {
	w.mu.Lock()
	w.backend = "playwright"
	w.fallbackOn = true
	w.mu.Unlock()

	w.page.On("framenavigated", func(frame playwright.Frame) {
		w.safeHandle(id, func() {
			if frame == nil {
				return
			}
			parent := frame.ParentFrame()
			if parent != nil {
				return
			}
			atMS := w.cfg.Now().UnixMilli()
			from := w.startedURL
			w.mu.Lock()
			from = w.lastURL
			w.mu.Unlock()
			ev := result.PageEvent{
				AtMS:    atMS,
				Kind:    "navigation",
				URL:     frame.URL(),
				FromURL: from,
				Frame:   "main",
			}
			w.appendEvent(ev)
		})
	})
	w.page.On("pageerror", func(err error) {
		w.safeHandle(id, func() {
			msg := ""
			if err != nil {
				msg = err.Error()
			}
			w.appendEvent(result.PageEvent{
				AtMS:    w.cfg.Now().UnixMilli(),
				Kind:    "pageerror",
				URL:     w.page.URL(),
				Message: truncateRunes(msg, maxMessageRunes),
				Frame:   "main",
			})
		})
	})
	w.page.On("crash", func() {
		w.safeHandle(id, func() {
			w.appendEvent(result.PageEvent{
				AtMS:  w.cfg.Now().UnixMilli(),
				Kind:  "crash",
				URL:   w.page.URL(),
				Frame: "main",
			})
		})
	})
	w.page.On("requestfailed", func(req playwright.Request) {
		w.safeHandle(id, func() {
			if req == nil {
				return
			}
			if !strings.EqualFold(req.ResourceType(), "document") {
				return
			}
			errText := ""
			if req.Failure() != nil {
				errText = req.Failure().Error()
			}
			w.appendEvent(result.PageEvent{
				AtMS:    w.cfg.Now().UnixMilli(),
				Kind:    "requestfailed",
				URL:     req.URL(),
				Message: truncateRunes(errText, maxMessageRunes),
				Frame:   "main",
			})
		})
	})
	return nil
}

func (w *Watch) safeHandle(id uint64, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			w.log.Error("pagewatch handler panic", "recover", r)
		}
	}()
	w.mu.Lock()
	active := w.activeID
	stopped := w.stopped
	w.mu.Unlock()
	if stopped || id != active {
		return
	}
	fn()
}

// Stop detaches CDP and marks handlers inactive. Idempotent.
func (w *Watch) Stop() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil
	}
	w.stopped = true
	w.watchID++
	w.activeID = w.watchID
	sess := w.session
	w.session = nil
	w.mu.Unlock()
	if sess != nil {
		if err := sess.Detach(); err != nil {
			return err
		}
	}
	return nil
}

// Snapshot copies current observation state.
func (w *Watch) Snapshot() result.PageWatch {
	if w == nil {
		return result.PageWatch{}
	}
	w.mu.Lock()
	events := append([]result.PageEvent(nil), w.events...)
	started := w.startedURL
	ended := w.lastURL
	trunc := w.truncated
	backend := w.backend
	page := w.page
	w.mu.Unlock()

	if page != nil {
		ended = page.URL()
	}
	snap := result.PageWatch{
		StartedURL:      started,
		EndedURL:        ended,
		Backend:         backend,
		Events:          events,
		EventsTruncated: trunc,
		ResetDetected:   ResetDetected(started, ended, events),
		ErrorDetected:   ErrorDetected(events),
	}
	return snap
}

func urlsMatch(a, b string) bool {
	return NormalizeURL(a) == NormalizeURL(b)
}
