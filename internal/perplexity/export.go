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
	URL       string
	ThreadID  string
	User      string
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
	path := ExportTargetPath(dir, threadID, "scrape")
	if err := os.WriteFile(path, []byte(markdown), 0o600); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	return abs, nil
}

// ExportTargetPath builds a stable markdown path under dir.
func ExportTargetPath(dir, threadID, label string) string {
	id := threadID
	if id == "" {
		id = "thr_unknown"
	}
	lab := sanitizeFilePart(label)
	if lab == "" {
		lab = "export"
	}
	name := time.Now().UTC().Format("20060102T150405Z") + "-" + sanitizeFilePart(id) + "-" + lab + ".md"
	return filepath.Join(dir, name)
}

// EnsureMarkdownExt forces a .md suffix (Playwright downloads often use a raw GUID name).
func EnsureMarkdownExt(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "export.md"
	}
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".markdown"), strings.HasSuffix(lower, ".txt"):
		if strings.HasSuffix(lower, ".txt") {
			return strings.TrimSuffix(name, filepath.Ext(name)) + ".md"
		}
		return name
	default:
		// GUID / extensionless Deep Research downloads.
		return name + ".md"
	}
}

// TryUIExportMarkdown attempts ⋯-menu / Deep Research download UI and SaveAs into exportDir.
// Export as Markdown lives under the thread three-dots menu next to Share (not under Share).
// Perplexity (especially over CDP) saves downloads as random GUID filenames; we must use
// ExpectDownload + SaveAs rather than polling ExportDir for *.md.
func TryUIExportMarkdown(page playwright.Page, exportDir, threadID string) (string, error) {
	if page == nil {
		return "", fmt.Errorf("no page")
	}
	if err := os.MkdirAll(exportDir, 0o700); err != nil {
		return "", err
	}
	_ = dismissCookieBannerSafe(page)

	var errs []string

	if p, err := tryMoreMenuDownload(page, exportDir, threadID); err == nil && p != "" {
		return p, nil
	} else if err != nil {
		errs = append(errs, "more_menu: "+err.Error())
	}

	if p, err := tryDeepResearchDownload(page, exportDir, threadID); err == nil && p != "" {
		return p, nil
	} else if err != nil {
		errs = append(errs, "deep_research: "+err.Error())
	}

	// Longer poll so a manual ⋯ → Export into ExportDir can still be picked up.
	if p := waitForNewestExportFile(exportDir, 8*time.Second); p != "" {
		return NormalizeExportFile(p, exportDir, threadID, "poll")
	}

	if len(errs) == 0 {
		return "", fmt.Errorf("markdown export UI path not completed")
	}
	return "", fmt.Errorf("markdown export UI path not completed (%s)", strings.Join(errs, "; "))
}

func tryMoreMenuDownload(page playwright.Page, exportDir, threadID string) (string, error) {
	if err := openMoreOptionsMenu(page); err != nil {
		return "", err
	}
	// Give the menu a moment to mount.
	time.Sleep(350 * time.Millisecond)

	for _, label := range ExportMenuPatterns {
		loc := findExportControl(page, label)
		if loc == nil {
			continue
		}
		if err := waitLocatorVisible(loc, 2_500); err != nil {
			continue
		}
		path, err := clickExpectSaveDownload(page, loc, exportDir, threadID, "more-menu")
		if err == nil && path != "" {
			return path, nil
		}
	}
	// Close menu so Deep Research controls stay clickable.
	_ = page.Keyboard().Press("Escape")
	return "", fmt.Errorf("no more-menu download completed")
}

func openMoreOptionsMenu(page playwright.Page) error {
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(5_000)}

	for _, re := range MoreOptionsNamePatterns {
		loc := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: re})
		if n, _ := loc.Count(); n > 0 {
			if err := loc.First().Click(force); err == nil {
				return nil
			}
		}
	}
	for _, sel := range MoreOptionsAriaSelectors {
		loc := page.Locator(sel)
		if n, _ := loc.Count(); n == 0 {
			continue
		}
		// Prefer toolbar control near Share when several aria-haspopup menus exist.
		if err := loc.First().Click(force); err == nil {
			return nil
		}
	}

	share := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: ShareButtonPattern})
	if n, err := share.Count(); err != nil || n == 0 {
		return fmt.Errorf("more options control not found")
	}
	// ⋯ is the button immediately left of Share in the thread toolbar.
	for _, xp := range []string{
		`xpath=preceding-sibling::button[1]`,
		`xpath=preceding::button[1]`,
	} {
		near := share.First().Locator(xp)
		if n, _ := near.Count(); n > 0 {
			if err := near.First().Click(force); err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("more options control not found (⋯ next to Share)")
}

func tryDeepResearchDownload(page playwright.Page, exportDir, threadID string) (string, error) {
	for _, label := range DeepResearchDownloadPatterns {
		loc := findExportControl(page, label)
		if loc == nil {
			continue
		}
		if err := waitLocatorVisible(loc, 1_500); err != nil {
			continue
		}
		path, err := clickExpectSaveDownload(page, loc, exportDir, threadID, "deep-research")
		if err == nil && path != "" {
			return path, nil
		}
	}
	return "", fmt.Errorf("no deep research download control completed")
}

func findExportControl(page playwright.Page, label string) playwright.Locator {
	re := regexp.MustCompile(label)
	roles := []playwright.AriaRole{
		*playwright.AriaRoleMenuitem,
		*playwright.AriaRoleButton,
		*playwright.AriaRoleLink,
	}
	for _, role := range roles {
		opt := page.GetByRole(role, playwright.PageGetByRoleOptions{Name: re})
		if cn, _ := opt.Count(); cn > 0 {
			return opt
		}
	}
	opt := page.GetByText(re)
	if cn, _ := opt.Count(); cn > 0 {
		return opt
	}
	return nil
}

func clickExpectSaveDownload(page playwright.Page, loc playwright.Locator, exportDir, threadID, label string) (string, error) {
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(5_000)}
	dl, err := page.ExpectDownload(func() error {
		return loc.First().Click(force)
	}, playwright.PageExpectDownloadOptions{
		Timeout: playwright.Float(20_000),
	})
	if err != nil {
		return "", err
	}
	return saveDownloadAsMarkdown(dl, exportDir, threadID, label)
}

func saveDownloadAsMarkdown(dl playwright.Download, exportDir, threadID, label string) (string, error) {
	if dl == nil {
		return "", fmt.Errorf("nil download")
	}
	suggested := dl.SuggestedFilename()
	// Prefer our stable name; keep extension hint from suggested when useful.
	baseLabel := label
	if suggested != "" && !looksLikeGUID(suggested) {
		baseLabel = strings.TrimSuffix(filepath.Base(suggested), filepath.Ext(suggested))
		if baseLabel == "" {
			baseLabel = label
		}
	}
	target := ExportTargetPath(exportDir, threadID, baseLabel)
	if err := dl.SaveAs(target); err != nil {
		return "", fmt.Errorf("save download: %w", err)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return target, nil
	}
	return abs, nil
}

func looksLikeGUID(name string) bool {
	name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	re := regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return re.MatchString(name)
}

func waitForNewestExportFile(dir string, timeout time.Duration) string {
	if timeout <= 0 {
		return newestExportFile(dir)
	}
	deadline := time.Now().Add(timeout)
	for {
		if p := newestExportFile(dir); p != "" {
			return p
		}
		if time.Now().After(deadline) {
			return ""
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func newestExportFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var best string
	var bestMod time.Time
	cutoff := time.Now().Add(-2 * time.Minute)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		// Accept .md / .markdown / .txt and extensionless GUID downloads.
		ok := strings.HasSuffix(lower, ".md") ||
			strings.HasSuffix(lower, ".markdown") ||
			strings.HasSuffix(lower, ".txt") ||
			!strings.Contains(name, ".")
		if !ok {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() < 32 {
			continue
		}
		if info.ModTime().Before(cutoff) {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best = filepath.Join(dir, name)
			bestMod = info.ModTime()
		}
	}
	return best
}

// NormalizeExportFile copies extensionless / non-md downloads into a stable .md path.
func NormalizeExportFile(src, exportDir, threadID, label string) (string, error) {
	if src == "" {
		return "", fmt.Errorf("empty path")
	}
	lower := strings.ToLower(src)
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") {
		abs, err := filepath.Abs(src)
		if err != nil {
			return src, nil
		}
		return abs, nil
	}
	body, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	target := ExportTargetPath(exportDir, threadID, label)
	if err := os.WriteFile(target, body, 0o600); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return target, nil
	}
	return abs, nil
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
