package perplexity

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

var (
	noiseLine = regexp.MustCompile(`(?i)^(answer|links|images|share|sources|ask a follow-up|search|computer|model|related|images?)$`)
)

// Conversation is a scraped thread for export.
type Conversation struct {
	URL      string
	ThreadID string
	User     string
	Assistant string
	Citations []result.Citation
	Turns     int
}

// ScrapeConversation builds a structured read of the current thread page.
func ScrapeConversation(page playwright.Page, threadID string) (Conversation, error) {
	out := Conversation{ThreadID: threadID}
	if page == nil {
		return out, fmt.Errorf("no page")
	}
	out.URL = page.URL()

	raw, err := ExtractAnswerText(page)
	if err != nil {
		return out, err
	}
	userPrompt, assistant := splitPromptAndAnswer(raw)
	out.User = cleanExportedText(userPrompt)
	out.Assistant = cleanExportedText(assistant)
	if out.Assistant == "" {
		out.Assistant = cleanExportedText(raw)
	}
	out.Citations = ExtractCitations(page)
	out.Turns = 0
	if out.User != "" {
		out.Turns++
	}
	if out.Assistant != "" {
		out.Turns++
	}
	if out.Turns == 0 {
		return out, fmt.Errorf("no conversation text found on page")
	}
	return out, nil
}

// FormatMarkdown renders a conversation export body.
func FormatMarkdown(c Conversation) string {
	var b strings.Builder
	b.WriteString("# Perplexity export\n")
	b.WriteString("URL: ")
	b.WriteString(c.URL)
	b.WriteString("\nThread: ")
	b.WriteString(c.ThreadID)
	b.WriteString("\nExported: ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("\n\n")

	turn := 1
	if c.User != "" {
		fmt.Fprintf(&b, "## Turn %d (user)\n\n%s\n\n", turn, c.User)
		turn++
	}
	if c.Assistant != "" {
		fmt.Fprintf(&b, "## Turn %d (assistant)\n\n%s\n\n", turn, c.Assistant)
		if len(c.Citations) > 0 {
			b.WriteString("### Citations\n")
			for _, cit := range c.Citations {
				title := cit.Title
				if title == "" {
					title = cit.URL
				}
				fmt.Fprintf(&b, "- [%s](%s)\n", title, cit.URL)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// WriteExportFile writes markdown under dir and returns the absolute path.
func WriteExportFile(dir, threadID, markdown string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	id := threadID
	if id == "" {
		id = "thr_unknown"
	}
	name := time.Now().UTC().Format("20060102T150405Z") + "-" + sanitizeFilePart(id) + ".md"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(markdown), 0o600); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	return abs, nil
}

// TryUIExportMarkdown attempts Share / export UI; returns empty path if unavailable.
func TryUIExportMarkdown(page playwright.Page, exportDir string) (string, error) {
	if page == nil {
		return "", fmt.Errorf("no page")
	}
	_ = dismissCookieBannerSafe(page)

	share := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: ShareButtonPattern})
	n, err := share.Count()
	if err != nil || n == 0 {
		return "", fmt.Errorf("share control not found")
	}
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(5_000)}
	if err := share.First().Click(force); err != nil {
		return "", err
	}

	for _, label := range ExportMenuPatterns {
		opt := page.GetByRole("menuitem", playwright.PageGetByRoleOptions{Name: regexp.MustCompile(label)})
		cn, _ := opt.Count()
		if cn == 0 {
			opt = page.GetByText(regexp.MustCompile(label))
			cn, _ = opt.Count()
		}
		if cn == 0 {
			continue
		}
		if err := waitLocatorVisible(opt, 3_000); err != nil {
			continue
		}
		if err := opt.First().Click(force); err != nil {
			continue
		}
		if p := waitForNewestMarkdown(exportDir, 4*time.Second); p != "" {
			return p, nil
		}
	}
	return "", fmt.Errorf("markdown export UI path not completed")
}

func waitForNewestMarkdown(dir string, timeout time.Duration) string {
	if timeout <= 0 {
		return newestMarkdown(dir)
	}
	deadline := time.Now().Add(timeout)
	for {
		if p := newestMarkdown(dir); p != "" {
			return p
		}
		if time.Now().After(deadline) {
			return ""
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func newestMarkdown(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var best string
	var bestMod time.Time
	cutoff := time.Now().Add(-2 * time.Minute)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best = filepath.Join(dir, e.Name())
			bestMod = info.ModTime()
		}
	}
	if best == "" {
		return ""
	}
	abs, err := filepath.Abs(best)
	if err != nil {
		return best
	}
	return abs
}

func splitPromptAndAnswer(raw string) (user, assistant string) {
	lines := strings.Split(raw, "\n")
	// Heuristic: first non-noise block is user, rest is assistant after a blank or after prompt echo.
	var buf []string
	phase := "lead"
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if noiseLine.MatchString(trim) {
			continue
		}
		if phase == "lead" {
			if trim == "" {
				if len(buf) > 0 {
					user = strings.TrimSpace(strings.Join(buf, "\n"))
					buf = nil
					phase = "answer"
				}
				continue
			}
			buf = append(buf, line)
			continue
		}
		buf = append(buf, line)
	}
	if phase == "lead" {
		// Single block: treat as assistant if it looks like an answer.
		assistant = strings.TrimSpace(strings.Join(buf, "\n"))
		return "", assistant
	}
	assistant = strings.TrimSpace(strings.Join(buf, "\n"))
	return user, assistant
}

func cleanExportedText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var keep []string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if noiseLine.MatchString(trim) {
			continue
		}
		keep = append(keep, line)
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

func sanitizeFilePart(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

func dismissCookieBannerSafe(page playwright.Page) error {
	dismissCookieBanner(page)
	return nil
}
