// Package perplexity automates perplexity.ai UI flows (mode, submit, wait, extract).
package perplexity

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/xynova/perplexity-browser/internal/result"
)

// ModeDeep is Perplexity Deep research.
const ModeDeep = "deep"

// ModeSearch is ordinary Search.
const ModeSearch = "search"

var stopLabel = regexp.MustCompile(`(?i)\b(stop|cancel)\b`)

// ErrUIChanged means selectors no longer match the live UI.
type ErrUIChanged struct {
	Op  string
	Msg string
}

func (e *ErrUIChanged) Error() string {
	return fmt.Sprintf("ui_changed [%s]: %s", e.Op, e.Msg)
}

// GotoHome navigates to the Perplexity home URL.
func GotoHome(page playwright.Page, baseURL string) error {
	if page == nil {
		return fmt.Errorf("no page")
	}
	if _, err := page.Goto(baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60_000),
	}); err != nil {
		return fmt.Errorf("goto home: %w", err)
	}
	time.Sleep(1200 * time.Millisecond)
	return nil
}

// EnsureCompose focuses a fresh compose box (new thread when possible).
func EnsureCompose(page playwright.Page) error {
	if page == nil {
		return fmt.Errorf("no page")
	}
	// Prefer explicit New Thread / New controls when present.
	for _, label := range []string{"New Thread", "New thread", "New"} {
		loc := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: label})
		n, err := loc.Count()
		if err == nil && n > 0 {
			vis, _ := loc.First().IsVisible()
			if vis {
				if err := loc.First().Click(); err == nil {
					time.Sleep(800 * time.Millisecond)
					break
				}
			}
		}
	}
	box, err := composeBox(page)
	if err != nil {
		return &ErrUIChanged{Op: "compose", Msg: err.Error()}
	}
	_ = box.Click()
	return nil
}

func composeBox(page playwright.Page) (playwright.Locator, error) {
	candidates := []string{
		`div[contenteditable="true"]`,
		`[role="textbox"]`,
		`textarea`,
	}
	for _, sel := range candidates {
		loc := page.Locator(sel)
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		first := loc.First()
		vis, _ := first.IsVisible()
		if vis {
			return first, nil
		}
	}
	return nil, fmt.Errorf("compose textbox not found")
}

// SetMode selects Search or Deep research on the compose controls.
func SetMode(page playwright.Page, mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = ModeDeep
	}
	if mode != ModeDeep && mode != ModeSearch {
		return fmt.Errorf("mode must be %q or %q", ModeDeep, ModeSearch)
	}

	// Open mode menu: button near Search / Research / Deep.
	opened := false
	for _, name := range []string{"Search", "Research", "Deep research", "Deep Research", "Pro"} {
		btn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: regexp.MustCompile("(?i)" + regexp.QuoteMeta(name))})
		n, err := btn.Count()
		if err != nil || n == 0 {
			continue
		}
		first := btn.First()
		vis, _ := first.IsVisible()
		if !vis {
			continue
		}
		if err := first.Click(); err == nil {
			opened = true
			time.Sleep(500 * time.Millisecond)
			break
		}
	}
	if !opened {
		// Try chevron / menuitem trigger via text.
		chev := page.GetByText(regexp.MustCompile(`(?i)(search|deep research|research)`))
		n, _ := chev.Count()
		if n > 0 {
			_ = chev.First().Click()
			time.Sleep(500 * time.Millisecond)
			opened = true
		}
	}
	if !opened {
		return &ErrUIChanged{Op: "mode_open", Msg: "could not open Search/Deep research mode control"}
	}

	var target *regexp.Regexp
	if mode == ModeDeep {
		target = regexp.MustCompile(`(?i)deep\s*research|^research$`)
	} else {
		target = regexp.MustCompile(`(?i)^search$`)
	}

	opt := page.GetByRole("menuitem", playwright.PageGetByRoleOptions{Name: target})
	n, err := opt.Count()
	if err != nil || n == 0 {
		// Fallback: any clickable text matching the mode.
		opt = page.GetByText(target)
		n, err = opt.Count()
		if err != nil || n == 0 {
			return &ErrUIChanged{Op: "mode_select", Msg: fmt.Sprintf("option for mode %q not found", mode)}
		}
	}
	// Prefer Deep research over bare Research when both match.
	chosen := opt.First()
	if mode == ModeDeep {
		deep := page.GetByText(regexp.MustCompile(`(?i)deep\s*research`))
		dn, _ := deep.Count()
		if dn > 0 {
			chosen = deep.First()
		}
	}
	if err := chosen.Click(); err != nil {
		return &ErrUIChanged{Op: "mode_click", Msg: err.Error()}
	}
	time.Sleep(400 * time.Millisecond)
	return nil
}

// SubmitPrompt types the prompt into compose and submits (Enter).
func SubmitPrompt(page playwright.Page, prompt string) error {
	box, err := composeBox(page)
	if err != nil {
		return &ErrUIChanged{Op: "compose", Msg: err.Error()}
	}
	if err := box.Click(); err != nil {
		return fmt.Errorf("focus compose: %w", err)
	}
	if err := box.Fill(""); err != nil {
		// contenteditable may not support Fill; fall through to type.
		_ = err
	}
	if err := box.PressSequentially(prompt, playwright.LocatorPressSequentiallyOptions{
		Delay: playwright.Float(2),
	}); err != nil {
		// Fallback: fill/type
		if err2 := box.Fill(prompt); err2 != nil {
			return fmt.Errorf("type prompt: %w (fill: %v)", err, err2)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if err := box.Press("Enter"); err != nil {
		// Try submit button
		submit := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: regexp.MustCompile(`(?i)(submit|send|ask)`)})
		n, _ := submit.Count()
		if n == 0 {
			return fmt.Errorf("submit: %w", err)
		}
		if err2 := submit.First().Click(); err2 != nil {
			return fmt.Errorf("submit enter: %w; click: %v", err, err2)
		}
	}
	return nil
}

// WaitComplete polls until generation finishes or ctx/timeout ends.
func WaitComplete(ctx context.Context, page playwright.Page, pollMS, timeoutMS int) error {
	if pollMS <= 0 {
		pollMS = 2000
	}
	if timeoutMS <= 0 {
		timeoutMS = 900_000
	}
	deadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	stable := 0
	var lastLen int

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		if time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		generating, err := isGenerating(page)
		if err != nil {
			return err
		}
		ans, _ := ExtractAnswerText(page)
		cur := len(ans)

		if generating {
			stable = 0
			lastLen = cur
		} else {
			if cur > 0 && cur == lastLen {
				stable++
			} else {
				stable = 0
				lastLen = cur
			}
			if stable >= 2 {
				return nil
			}
		}
		time.Sleep(time.Duration(pollMS) * time.Millisecond)
	}
}

func isGenerating(page playwright.Page) (bool, error) {
	// Stop / Cancel generation button visible.
	btns := page.GetByRole("button")
	n, err := btns.Count()
	if err != nil {
		return false, err
	}
	limit := n
	if limit > 40 {
		limit = 40
	}
	for i := 0; i < limit; i++ {
		btn := btns.Nth(i)
		vis, _ := btn.IsVisible()
		if !vis {
			continue
		}
		name, _ := btn.GetAttribute("aria-label")
		text, _ := btn.InnerText()
		blob := name + " " + text
		if stopLabel.MatchString(blob) && !strings.Contains(strings.ToLower(blob), "account") {
			return true, nil
		}
	}
	// Progress text.
	prog := page.GetByText(regexp.MustCompile(`(?i)(researching|thinking|searching|generating)`))
	pn, _ := prog.Count()
	if pn > 0 {
		vis, _ := prog.First().IsVisible()
		if vis {
			return true, nil
		}
	}
	return false, nil
}

// ExtractAnswerText returns best-effort latest assistant prose.
func ExtractAnswerText(page playwright.Page) (string, error) {
	if page == nil {
		return "", fmt.Errorf("no page")
	}
	// Prefer main article / prose regions.
	sels := []string{
		`[data-testid="answer"]`,
		`main article`,
		`main .prose`,
		`[class*="answer"]`,
		`main`,
	}
	var best string
	for _, sel := range sels {
		loc := page.Locator(sel)
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		text, err := loc.Last().InnerText()
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if len(text) > len(best) {
			best = text
		}
	}
	return best, nil
}

// ExtractCitations collects link-looking citations from the page.
func ExtractCitations(page playwright.Page) []result.Citation {
	var out []result.Citation
	links := page.Locator(`main a[href^="http"]`)
	n, err := links.Count()
	if err != nil || n == 0 {
		return out
	}
	limit := n
	if limit > 30 {
		limit = 30
	}
	seen := map[string]bool{}
	for i := 0; i < limit; i++ {
		a := links.Nth(i)
		href, err := a.GetAttribute("href")
		if err != nil || href == "" || seen[href] {
			continue
		}
		if strings.Contains(href, "perplexity.ai") {
			continue
		}
		title, _ := a.InnerText()
		title = strings.TrimSpace(title)
		seen[href] = true
		out = append(out, result.Citation{Title: title, URL: href})
	}
	return out
}

// Truncate caps answer length for tool responses.
func Truncate(s string, max int) (string, bool) {
	if max <= 0 || len(s) <= max {
		return s, false
	}
	return s[:max] + "\n…[truncated]", true
}
