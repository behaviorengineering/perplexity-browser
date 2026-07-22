package session

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/behaviorengineering/perplexity-browser/internal/config"
)

const cdpReadyTimeout = 45 * time.Second

func (m *Manager) connectCDPLocked() error {
	err := m.tryConnectCDPLocked()
	if err == nil {
		return nil
	}
	if !m.cfg.CDPAutoLaunch {
		return fmt.Errorf("connect over CDP %s: %w (set PERPLEXITY_BROWSER_CDP_AUTO_LAUNCH=1 or run scripts/chrome-cdp.sh)", m.cfg.CDPURL, err)
	}

	m.log.Info("CDP connect failed; attempting Chrome launch", "cdp_url", m.cfg.CDPURL, "err", err)

	if waitErr := waitCDPReady(context.Background(), m.cfg.CDPURL, 2*time.Second); waitErr == nil {
		if retryErr := m.tryConnectCDPLocked(); retryErr == nil {
			return nil
		}
	}

	if launchErr := launchChromeForCDP(m.cfg, m.log); launchErr != nil {
		return fmt.Errorf("connect over CDP %s: %w; auto-launch failed: %v (quit other Chrome using profile %q, or run scripts/chrome-cdp.sh)", m.cfg.CDPURL, err, launchErr, m.cfg.UserDataDir)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), cdpReadyTimeout)
	defer cancel()
	if waitErr := waitCDPReady(waitCtx, m.cfg.CDPURL, cdpReadyTimeout); waitErr != nil {
		return fmt.Errorf("connect over CDP %s: %w; Chrome launched but CDP not ready: %v", m.cfg.CDPURL, err, waitErr)
	}

	if retryErr := m.tryConnectCDPLocked(); retryErr != nil {
		return fmt.Errorf("connect over CDP %s after auto-launch: %w", m.cfg.CDPURL, retryErr)
	}
	m.log.Info("connected over CDP after auto-launch", "cdp_url", m.cfg.CDPURL)
	return nil
}

func (m *Manager) tryConnectCDPLocked() error {
	b, err := m.pw.Chromium.ConnectOverCDP(m.cfg.CDPURL)
	if err != nil {
		return err
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
		m.page = pickBestPage(pages)
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
	m.log.Info("connected over CDP", "cdp_url", m.cfg.CDPURL, "pages", len(m.browser.Pages()))
	return nil
}

func launchChromeForCDP(cfg config.Config, log interface {
	Info(msg string, args ...any)
}) error {
	if err := os.MkdirAll(cfg.UserDataDir, 0o700); err != nil {
		return fmt.Errorf("create user data dir: %w", err)
	}
	if cfg.ChromeApp == "" {
		return fmt.Errorf("chrome binary path is empty")
	}
	if _, err := os.Stat(cfg.ChromeApp); err != nil {
		return fmt.Errorf("chrome binary %q: %w", cfg.ChromeApp, err)
	}

	port, err := cdpPortFromURL(cfg.CDPURL)
	if err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", cfg.UserDataDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate",
		cfg.BaseURL,
	}

	cmd := exec.Command(cfg.ChromeApp, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start chrome: %w", err)
	}
	log.Info("launched Chrome for CDP", "chrome", cfg.ChromeApp, "port", port, "profile", cfg.UserDataDir)
	return nil
}

func cdpPortFromURL(raw string) (int, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("parse CDP URL %q: %w", raw, err)
	}
	host := u.Host
	if host == "" {
		return 0, fmt.Errorf("CDP URL %q has no host", raw)
	}
	_, portStr, err := net.SplitHostPort(host)
	if err != nil {
		// Default CDP port when host has no explicit port.
		if strings.Contains(err.Error(), "missing port") {
			return 9222, nil
		}
		return 0, fmt.Errorf("CDP host %q: %w", host, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid CDP port %q", portStr)
	}
	return port, nil
}

func waitCDPReady(ctx context.Context, cdpURL string, maxWait time.Duration) error {
	if maxWait <= 0 {
		maxWait = cdpReadyTimeout
	}
	deadline := time.Now().Add(maxWait)
	client := &http.Client{Timeout: 2 * time.Second}
	checkURL := strings.TrimRight(cdpURL, "/") + "/json/version"
	poll := 400 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("CDP not ready at %s within %s", checkURL, maxWait)
		}
		time.Sleep(poll)
	}
}
