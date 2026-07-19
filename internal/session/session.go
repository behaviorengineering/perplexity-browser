// Package session owns the Playwright persistent browser and thread state.
package session

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/xynova/perplexity-browser/internal/config"
	"github.com/xynova/perplexity-browser/internal/result"
)

// Manager is the single browser + thread owner for one MCP process.
type Manager struct {
	cfg    config.Config
	log    *slog.Logger
	mu     sync.Mutex
	busy   bool
	runCtx context.Context
	cancel context.CancelFunc

	pw         *playwright.Playwright
	cdpBrowser playwright.Browser // set when connected over CDP
	browser    playwright.BrowserContext
	page       playwright.Page
	cdpMode    bool

	threadID string
	mode     string
}

// New creates a Manager (browser not started until EnsureBrowser).
func New(cfg config.Config, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{cfg: cfg, log: log}
}

// Config returns a copy of runtime config.
func (m *Manager) Config() config.Config { return m.cfg }

// TryBegin marks the session busy for a mutating tool. Caller must End.
func (m *Manager) TryBegin() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy {
		return false
	}
	m.busy = true
	m.runCtx, m.cancel = context.WithCancel(context.Background())
	return true
}

// End clears the busy flag.
func (m *Manager) End() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.busy = false
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.runCtx = nil
}

// Cancel aborts an in-flight wait without closing the browser.
func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
}

// IsBusy reports whether a mutating tool is running.
func (m *Manager) IsBusy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.busy
}

// EnsureBrowser starts Playwright with a persistent user-data dir if needed.
func (m *Manager) EnsureBrowser(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureBrowserLocked(ctx)
}

func (m *Manager) ensureBrowserLocked(ctx context.Context) error {
	_ = ctx
	if m.page != nil {
		if alive := m.pageAliveLocked(); alive {
			return nil
		}
		m.log.Info("browser context dead; reconnecting", "cdp", m.cfg.CDPURL != "", "user_data_dir", m.cfg.UserDataDir)
		m.dropContextLocked()
	}
	if err := os.MkdirAll(m.cfg.ExportDir, 0o700); err != nil {
		return fmt.Errorf("create export dir: %w", err)
	}

	if m.pw == nil {
		pw, err := playwright.Run()
		if err != nil {
			return fmt.Errorf("playwright run: %w", err)
		}
		m.pw = pw
	}

	if m.cfg.CDPURL != "" {
		return m.connectCDPLocked()
	}
	return m.launchPersistentLocked()
}

func (m *Manager) connectCDPLocked() error {
	b, err := m.pw.Chromium.ConnectOverCDP(m.cfg.CDPURL)
	if err != nil {
		return fmt.Errorf("connect over CDP %s: %w (start Chrome with scripts/chrome-cdp.sh first)", m.cfg.CDPURL, err)
	}
	m.cdpBrowser = b
	m.cdpMode = true
	contexts := b.Contexts()
	if len(contexts) == 0 {
		_ = b.Close()
		m.cdpBrowser = nil
		m.cdpMode = false
		return fmt.Errorf("CDP browser has no contexts")
	}
	m.browser = contexts[0]
	pages := m.browser.Pages()
	if len(pages) > 0 {
		m.page = pages[0]
	} else {
		page, err := m.browser.NewPage()
		if err != nil {
			_ = b.Close()
			m.cdpBrowser = nil
			m.browser = nil
			m.cdpMode = false
			return fmt.Errorf("new page over CDP: %w", err)
		}
		m.page = page
	}
	m.log.Info("connected over CDP", "cdp_url", m.cfg.CDPURL, "pages", len(pages))
	return nil
}

func (m *Manager) launchPersistentLocked() error {
	if err := os.MkdirAll(m.cfg.UserDataDir, 0o700); err != nil {
		return fmt.Errorf("create user data dir: %w", err)
	}

	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless:        playwright.Bool(m.cfg.Headless),
		AcceptDownloads: playwright.Bool(true),
		DownloadsPath:   playwright.String(m.cfg.ExportDir),
		Viewport:        &playwright.Size{Width: 1280, Height: 900},
	}
	if m.cfg.Channel != "" {
		opts.Channel = playwright.String(m.cfg.Channel)
	}

	browser, err := m.pw.Chromium.LaunchPersistentContext(m.cfg.UserDataDir, opts)
	if err != nil {
		return fmt.Errorf("launch persistent context: %w", err)
	}
	m.browser = browser
	m.cdpMode = false

	pages := browser.Pages()
	if len(pages) > 0 {
		m.page = pages[0]
	} else {
		page, err := browser.NewPage()
		if err != nil {
			_ = browser.Close()
			m.browser = nil
			return fmt.Errorf("new page: %w", err)
		}
		m.page = page
	}
	m.log.Info("browser ready", "user_data_dir", m.cfg.UserDataDir, "headless", m.cfg.Headless, "channel", m.cfg.Channel)
	return nil
}

// dropContextLocked drops Playwright handles. In CDP mode it disconnects only;
// Chrome itself is left running. Persistent profile cookies stay on disk.
func (m *Manager) dropContextLocked() {
	m.threadID = ""
	m.mode = ""
	if m.cdpMode {
		if m.cdpBrowser != nil {
			_ = m.cdpBrowser.Close() // disconnect from CDP
			m.cdpBrowser = nil
		}
		m.browser = nil
		m.page = nil
		m.cdpMode = false
		return
	}
	if m.browser != nil {
		_ = m.browser.Close()
		m.browser = nil
	}
	m.page = nil
}

func (m *Manager) pageAliveLocked() bool {
	if m.page == nil {
		return false
	}
	_, err := m.page.Evaluate("() => true")
	if err == nil {
		return true
	}
	return !isTargetClosed(err)
}

func isTargetClosed(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "target closed") ||
		strings.Contains(s, "has been closed") ||
		strings.Contains(s, "browser has been closed") ||
		strings.Contains(s, "context or browser has been closed")
}

// CloseBrowser stops the browser context and Playwright driver.
func (m *Manager) CloseBrowser() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.busy = false
	m.threadID = ""
	m.mode = ""

	var first error
	if m.cdpMode {
		m.dropContextLocked()
	} else if m.browser != nil {
		if err := m.browser.Close(); err != nil && first == nil {
			first = err
		}
		m.browser = nil
		m.page = nil
	}
	if m.pw != nil {
		if err := m.pw.Stop(); err != nil && first == nil {
			first = err
		}
		m.pw = nil
	}
	return first
}

// Status returns session JSON fields without requiring the write lock long.
func (m *Manager) Status(ctx context.Context, openBrowser bool) result.Session {
	out := result.Session{
		Base: result.Base{
			Status: result.StatusOK,
			Busy:   m.IsBusy(),
		},
		UserDataDir: m.cfg.UserDataDir,
		ExportDir:   m.cfg.ExportDir,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	out.ThreadID = m.threadID
	if m.page != nil {
		if !m.pageAliveLocked() {
			m.log.Info("status: dead page; will relaunch persistent profile")
			m.dropContextLocked()
		} else {
			out.BrowserOpen = true
			out.URL = m.page.URL()
			loggedIn, err := detectLogin(m.page)
			if err != nil {
				if isTargetClosed(err) {
					m.dropContextLocked()
				} else {
					out.Message = err.Error()
					out.LoggedIn = false
					return out
				}
			} else {
				out.LoggedIn = loggedIn
				if !loggedIn {
					out.Status = result.StatusNeedLogin
					out.Message = "sign in in the headed browser window, then call wait_for_login or retry"
				}
				return out
			}
		}
	}

	if !openBrowser {
		out.BrowserOpen = false
		out.Message = "browser not open"
		return out
	}

	if err := m.ensureBrowserLocked(ctx); err != nil {
		out.Status = result.StatusError
		out.Message = err.Error()
		return out
	}
	// CDP: do not force navigation (avoids fresh bot challenges). Use the open tab.
	if m.cfg.CDPURL == "" {
		if err := m.gotoHomeLocked(ctx); err != nil {
			if isTargetClosed(err) {
				m.dropContextLocked()
				if err2 := m.ensureBrowserLocked(ctx); err2 != nil {
					out.Status = result.StatusError
					out.Message = err2.Error()
					return out
				}
				if err2 := m.gotoHomeLocked(ctx); err2 != nil {
					out.Status = result.StatusError
					out.Message = err2.Error()
					out.BrowserOpen = m.page != nil
					return out
				}
			} else {
				out.Status = result.StatusError
				out.Message = err.Error()
				out.BrowserOpen = m.page != nil
				return out
			}
		}
	}
	out.BrowserOpen = true
	out.URL = m.page.URL()
	loggedIn, err := detectLogin(m.page)
	if err != nil {
		out.Message = err.Error()
	} else {
		out.LoggedIn = loggedIn
		if !loggedIn {
			out.Status = result.StatusNeedLogin
			if botWall(m.page) {
				out.Message = "bot / security verification page detected; complete it in Chrome (CDP), then retry status"
			} else if m.cfg.CDPURL != "" {
				out.Message = "not logged in on the CDP Chrome tab; sign in there, open perplexity.ai home, then retry status"
			} else {
				out.Message = "sign in in the headed browser window, then call wait_for_login or retry"
			}
		}
	}
	return out
}

// WaitForLogin polls until logged in or timeout.
func (m *Manager) WaitForLogin(ctx context.Context, timeoutMS int) result.Session {
	if timeoutMS <= 0 {
		timeoutMS = 600_000
	}
	deadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	poll := time.Duration(m.cfg.PollMS) * time.Millisecond
	if poll < 500*time.Millisecond {
		poll = 2 * time.Second
	}

	runCtx := m.runContext()
	for {
		select {
		case <-ctx.Done():
			s := m.Status(context.Background(), true)
			s.Status = result.StatusCancelled
			s.Message = "wait_for_login cancelled"
			return s
		case <-runCtx.Done():
			s := m.Status(context.Background(), true)
			s.Status = result.StatusCancelled
			s.Message = "wait_for_login cancelled"
			return s
		default:
		}

		s := m.Status(ctx, true)
		if s.LoggedIn && s.Status == result.StatusOK {
			s.Message = "logged in"
			return s
		}
		if time.Now().After(deadline) {
			s.Status = result.StatusTimeout
			s.Message = "wait_for_login timed out"
			return s
		}
		time.Sleep(poll)
	}
}

// GotoHome navigates to the configured base URL.
func (m *Manager) GotoHome(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureBrowserLocked(ctx); err != nil {
		return err
	}
	return m.gotoHomeLocked(ctx)
}

func (m *Manager) gotoHomeLocked(ctx context.Context) error {
	_ = ctx
	if m.page == nil {
		return fmt.Errorf("no page")
	}
	if _, err := m.page.Goto(m.cfg.BaseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60_000),
	}); err != nil {
		return fmt.Errorf("goto %s: %w", m.cfg.BaseURL, err)
	}
	// Give client hydration a moment for login UI.
	time.Sleep(1500 * time.Millisecond)
	return nil
}

// SetThread records the active thread metadata.
func (m *Manager) SetThread(id, mode, pageURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threadID = id
	m.mode = mode
	_ = pageURL
}

// ThreadID returns the active thread id.
func (m *Manager) ThreadID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.threadID
}

// PageURL returns the current page URL if the browser is open.
func (m *Manager) PageURL() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.page == nil {
		return ""
	}
	return m.page.URL()
}

func (m *Manager) runContext() context.Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runCtx == nil {
		return context.Background()
	}
	return m.runCtx
}

// detectLogin uses fail-closed heuristics from the technical appendix.
func detectLogin(page playwright.Page) (bool, error) {
	if page == nil {
		return false, fmt.Errorf("no page")
	}
	if botWall(page) {
		return false, nil
	}
	url := strings.ToLower(page.URL())
	if strings.Contains(url, "login") || strings.Contains(url, "signin") || strings.Contains(url, "auth") {
		return false, nil
	}

	loginHints := []string{
		"Sign in",
		"Log in",
		"Continue with Google",
		"Continue with Apple",
	}
	for _, text := range loginHints {
		loc := page.GetByText(text, playwright.PageGetByTextOptions{Exact: playwright.Bool(false)})
		count, err := loc.Count()
		if err != nil {
			continue
		}
		if count > 0 {
			vis, _ := loc.First().IsVisible()
			if vis {
				return false, nil
			}
		}
	}

	compose := page.Locator(`div[contenteditable="true"], textarea, [role="textbox"]`)
	n, err := compose.Count()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	return true, nil
}

func botWall(page playwright.Page) bool {
	if page == nil {
		return false
	}
	hints := []string{
		"security verification",
		"verify you are not a bot",
		"verifying you are human",
		"just a moment",
		"checking your browser",
		"attention required",
	}
	for _, h := range hints {
		loc := page.GetByText(h, playwright.PageGetByTextOptions{Exact: playwright.Bool(false)})
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		vis, _ := loc.First().IsVisible()
		if vis {
			return true
		}
	}
	return false
}
