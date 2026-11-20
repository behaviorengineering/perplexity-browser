package perplexity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

var noiseLine = regexp.MustCompile(`(?i)^(answer|links|images|share|sources|ask a follow-up|search|computer|model|related|images?)$`)

// Conversation is a scraped thread for export.
type Conversation struct {
	URL       string
	ThreadID  string
	User      string
	Assistant string
	Citations []result.Citation
	Turns     int
}

// ConversationFromTexts builds export conversation fields from separate user and assistant extracts.
func ConversationFromTexts(url, threadID, user, assistant string, cits []result.Citation) (Conversation, error) {
	out := Conversation{
		URL:       url,
		ThreadID:  threadID,
		User:      cleanExportedText(user),
		Assistant: cleanExportedText(assistant),
		Citations: cits,
	}
	userNorm := strings.TrimSpace(out.User)
	asstNorm := strings.TrimSpace(out.Assistant)
	if userNorm != "" {
		if userNorm == asstNorm || (asstNorm != "" && strings.HasPrefix(asstNorm, userNorm)) {
			out.User = ""
		}
	}
	out.Turns = 0
	if out.User != "" {
		out.Turns++
	}
	if out.Assistant != "" {
		out.Turns++
	}
	if out.Turns == 0 {
		return out, newErr("ConversationFromTexts", CodeInvalid, "no conversation text found on page", nil)
	}
	return out, nil
}

// ScrapeConversation builds a structured read of the current thread page.
func ScrapeConversation(page playwright.Page, threadID string) (Conversation, error) {
	out := Conversation{ThreadID: threadID}
	if page == nil {
		return out, newErr("ScrapeConversation", CodeInvalid, "no page", nil)
	}
	user, err := ExtractUserPrompt(page)
	if err != nil {
		return out, err
	}
	assistant, err := ExtractAnswerText(page)
	if err != nil {
		return out, err
	}
	cits := ExtractCitations(page)
	return ConversationFromTexts(page.URL(), threadID, user, assistant, cits)
}

const markdownTpl = `# Perplexity export
URL: {{.URL}}
Thread: {{.ThreadID}}
Exported: {{.Exported}}
{{if .User}}
## Turn {{.UserTurn}} (user)

{{.User}}
{{end}}{{if .Assistant}}
## Turn {{.AsstTurn}} (assistant)

{{.Assistant}}
{{if .Citations}}
### Citations
{{range .Citations}}- [{{citationTitle .}}]({{.URL}})
{{end}}{{end}}{{end}}`

var markdownTemplate = template.Must(template.New("export").Funcs(template.FuncMap{
	"citationTitle": func(c result.Citation) string {
		if c.Title != "" {
			return c.Title
		}
		return c.URL
	},
}).Parse(markdownTpl))

type markdownView struct {
	URL       string
	ThreadID  string
	Exported  string
	User      string
	Assistant string
	UserTurn  int
	AsstTurn  int
	Citations []result.Citation
}

// FormatMarkdown renders a conversation export body.
func FormatMarkdown(c Conversation) string {
	return formatMarkdownAt(c, time.Now().UTC())
}

func formatMarkdownAt(c Conversation, now time.Time) string {
	view := markdownView{
		URL:       c.URL,
		ThreadID:  c.ThreadID,
		Exported:  now.Format(time.RFC3339),
		User:      c.User,
		Assistant: c.Assistant,
		Citations: c.Citations,
	}
	turn := 1
	if c.User != "" {
		view.UserTurn = turn
		turn++
	}
	if c.Assistant != "" {
		view.AsstTurn = turn
	}
	var b strings.Builder
	if err := markdownTemplate.Execute(&b, view); err != nil {
		return ""
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
	return exportTargetPathAt(dir, threadID, label, time.Now().UTC())
}

func exportTargetPathAt(dir, threadID, label string, now time.Time) string {
	id := threadID
	if id == "" {
		id = "thr_unknown"
	}
	lab := sanitizeFilePart(label)
	if lab == "" {
		lab = "export"
	}
	name := now.UTC().Format("20060102T150405Z") + "-" + sanitizeFilePart(id) + "-" + lab + ".md"
	return filepath.Join(dir, name)
}

// EnsureMarkdownExt forces a .md suffix (Playwright downloads often use a raw GUID name).
func EnsureMarkdownExt(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "export.md"
	}
	lower := strings.ToLower(name)
	if LooksLikePDFName(name) {
		return name
	}
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
		return "", newErr("TryUIExportMarkdown", CodeInvalid, "no page", nil)
	}
	if err := os.MkdirAll(exportDir, 0o700); err != nil {
		return "", newErr("TryUIExportMarkdown", CodeUnavailable, "mkdir export dir", err)
	}
	_ = dismissCookieBannerSafe(page)

	var errs []error

	if p, err := tryMoreMenuDownload(page, exportDir, threadID); err == nil && p != "" {
		return p, nil
	} else if err != nil {
		errs = append(errs, fmt.Errorf("more_menu: %w", err))
	}

	if p, err := tryDeepResearchDownload(page, exportDir, threadID); err == nil && p != "" {
		return p, nil
	} else if err != nil {
		errs = append(errs, fmt.Errorf("deep_research: %w", err))
	}

	if len(errs) == 0 {
		return "", newErr("TryUIExportMarkdown", CodeUnavailable, "markdown export UI path not completed", nil)
	}
	return "", newErr("TryUIExportMarkdown", CodeUnavailable, "markdown export UI path not completed", errors.Join(errs...))
}

// MenuDownloadError joins the menu opener id and per-label attempt failures for diagnostics.
func MenuDownloadError(summary, opener string, attempts []error) error {
	var parts []error
	if opener != "" {
		parts = append(parts, fmt.Errorf("opener=%s", opener))
	}
	parts = append(parts, attempts...)
	return newErr("MenuDownloadError", CodeUnavailable, summary, errors.Join(parts...))
}

func tryMoreMenuDownload(page playwright.Page, exportDir, threadID string) (string, error) {
	opener, err := openMoreOptionsMenu(page)
	if err != nil {
		return "", err
	}
	// Give the menu a moment to mount.
	time.Sleep(350 * time.Millisecond)

	var attempts []error
	for _, label := range ExportMenuPatterns {
		loc := findExportControl(page, label)
		if loc == nil {
			attempts = append(attempts, fmt.Errorf("%s: not found", label))
			continue
		}
		if err := waitLocatorVisible(loc, 2_500); err != nil {
			attempts = append(attempts, fmt.Errorf("%s: not visible: %w", label, err))
			continue
		}
		path, err := clickExpectSaveDownload(page, loc, exportDir, threadID, "more-menu")
		if err == nil && path != "" {
			return path, nil
		}
		if err != nil {
			attempts = append(attempts, fmt.Errorf("%s: download: %w", label, err))
		} else {
			attempts = append(attempts, fmt.Errorf("%s: download: empty path", label))
		}
	}
	// Close menu so Deep Research controls stay clickable.
	_ = page.Keyboard().Press("Escape")
	return "", MenuDownloadError("no more-menu download completed", opener, attempts)
}

func openMoreOptionsMenu(page playwright.Page) (string, error) {
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(5_000)}
	var clickErrs []error

	for _, re := range MoreOptionsNamePatterns {
		loc := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: re})
		n, err := loc.Count()
		if err != nil {
			clickErrs = append(clickErrs, fmt.Errorf("count role=%s: %w", re.String(), err))
			continue
		}
		if n == 0 {
			continue
		}
		if err := loc.First().Click(force); err != nil {
			clickErrs = append(clickErrs, fmt.Errorf("click role=%s: %w", re.String(), err))
			continue
		}
		return "role=button name=" + re.String(), nil
	}
	for _, sel := range MoreOptionsAriaSelectors {
		loc := page.Locator(sel)
		n, err := loc.Count()
		if err != nil {
			clickErrs = append(clickErrs, fmt.Errorf("count %s: %w", sel, err))
			continue
		}
		if n == 0 {
			continue
		}
		if err := loc.First().Click(force); err != nil {
			clickErrs = append(clickErrs, fmt.Errorf("click %s: %w", sel, err))
			continue
		}
		return sel, nil
	}

	share := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: ShareButtonPattern})
	n, err := share.Count()
	if err != nil {
		return "", newErr("openMoreOptionsMenu", CodeUnavailable, "more options control not found", errors.Join(append(clickErrs, err)...))
	}
	if n == 0 {
		return "", newErr("openMoreOptionsMenu", CodeUnavailable, "more options control not found", errors.Join(clickErrs...))
	}
	// ⋯ is the button immediately left of Share in the thread toolbar.
	for _, xp := range []string{
		`xpath=preceding-sibling::button[1]`,
		`xpath=preceding::button[1]`,
	} {
		near := share.First().Locator(xp)
		n, err := near.Count()
		if err != nil {
			clickErrs = append(clickErrs, fmt.Errorf("count %s: %w", xp, err))
			continue
		}
		if n == 0 {
			continue
		}
		if err := near.First().Click(force); err != nil {
			clickErrs = append(clickErrs, fmt.Errorf("click %s: %w", xp, err))
			continue
		}
		return "share-toolbar:" + xp, nil
	}
	return "", newErr("openMoreOptionsMenu", CodeUnavailable, "more options control not found (next to Share)", errors.Join(clickErrs...))
}

func tryDeepResearchDownload(page playwright.Page, exportDir, threadID string) (string, error) {
	var attempts []error
	for _, label := range DeepResearchDownloadPatterns {
		loc := findExportControl(page, label)
		if loc == nil {
			attempts = append(attempts, fmt.Errorf("%s: not found", label))
			continue
		}
		if err := waitLocatorVisible(loc, 1_500); err != nil {
			attempts = append(attempts, fmt.Errorf("%s: not visible: %w", label, err))
			continue
		}
		path, err := clickExpectSaveDownload(page, loc, exportDir, threadID, "deep-research")
		if err == nil && path != "" {
			return path, nil
		}
		if err != nil {
			attempts = append(attempts, fmt.Errorf("%s: download: %w", label, err))
		} else {
			attempts = append(attempts, fmt.Errorf("%s: download: empty path", label))
		}
	}
	return "", MenuDownloadError("no deep research download control completed", "", attempts)
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
	if LooksLikePDFName(suggested) {
		return "", fmt.Errorf("download was PDF, not markdown (%s)", suggested)
	}
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
	body, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	if IsPDFBytes(body) {
		_ = os.Remove(target)
		return "", fmt.Errorf("download was PDF, not markdown (%s)", suggested)
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

// LooksLikePDFName reports a .pdf path or download filename.
func LooksLikePDFName(name string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), ".pdf")
}

// IsPDFBytes reports the PDF magic header.
func IsPDFBytes(b []byte) bool {
	return len(b) >= 5 && string(b[:5]) == "%PDF-"
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
	if LooksLikePDFName(src) {
		return "", fmt.Errorf("refusing PDF export %s", src)
	}
	body, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if IsPDFBytes(body) {
		return "", fmt.Errorf("refusing PDF export %s", src)
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
