// Package perplexity automates perplexity.ai UI flows (mode, submit, wait, extract).
package perplexity

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

// ModeDeep is Perplexity Deep research.
const ModeDeep = "deep"

// ModeSearch is ordinary Search.
const ModeSearch = "search"

var stopLabel = regexp.MustCompile(`(?i)\b(stop|cancel)\b`)

// regexpStopGenerating matches Stop / Stop generating controls (not Cancel).
var regexpStopGenerating = regexp.MustCompile(`(?i)stop(\s+generating|\s+research|\s+response)?`)


// ErrUIChanged means selectors no longer match the live UI.
type ErrUIChanged struct {
	Op  string
	Msg string
}

func (e *ErrUIChanged) Error() string {
	return fmt.Sprintf("ui_changed [%s]: %s", e.Op, e.Msg)
}

// GotoHome navigates to the Perplexity home URL, reloads once to clear stale
// collection/project prompts that sometimes appear on first paint, then settles.
func GotoHome(page playwright.Page, baseURL string) error {
	return GotoHomeViaWarmup(page, "", baseURL)
}

// GotoHomeViaWarmup optionally opens warmupURL first (e.g. google.com) when the
// tab is not already on Perplexity, then navigates to baseURL + reload settle.
func GotoHomeViaWarmup(page playwright.Page, warmupURL, baseURL string) error {
	if page == nil {
		return fmt.Errorf("no page")
	}
	if err := maybeWarmup(page, warmupURL, baseURL); err != nil {
		return err
	}
	if _, err := page.Goto(baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60_000),
	}); err != nil {
		return fmt.Errorf("goto home: %w", err)
	}
	if err := ReloadAfterNavigate(page); err != nil {
		return err
	}
	WaitAfterHome(page)
	DismissBlockingDialogs(page)
	return nil
}

// ShouldWarmup reports whether a cold-start hop through warmupURL is useful.
func ShouldWarmup(currentURL, warmupURL, baseURL string) bool {
	warmupURL = strings.TrimSpace(warmupURL)
	baseURL = strings.TrimSpace(baseURL)
	if warmupURL == "" || baseURL == "" {
		return false
	}
	cur := strings.ToLower(strings.TrimSpace(currentURL))
	if cur == "" || cur == "about:blank" || strings.HasPrefix(cur, "chrome://") {
		return true
	}
	if strings.Contains(cur, "perplexity.ai") {
		return false
	}
	// Already sitting on the warmup host — still "warm"; caller may skip re-goto.
	if warmupHostMatch(cur, warmupURL) {
		return false
	}
	return true
}

func maybeWarmup(page playwright.Page, warmupURL, baseURL string) error {
	_ = baseURL
	warmupURL = strings.TrimSpace(warmupURL)
	if warmupURL == "" || page == nil {
		return nil
	}
	cur := page.URL()
	if strings.Contains(strings.ToLower(cur), "perplexity.ai") {
		return nil
	}
	if warmupHostMatch(cur, warmupURL) {
		return nil
	}
	if _, err := page.Goto(warmupURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(45_000),
	}); err != nil {
		return fmt.Errorf("goto warmup %s: %w", warmupURL, err)
	}
	time.Sleep(400 * time.Millisecond)
	return nil
}

func warmupHostMatch(currentURL, warmupURL string) bool {
	cur := strings.ToLower(currentURL)
	w := strings.ToLower(warmupURL)
	// Cheap contains check for google.com / configured host.
	for _, host := range []string{"google.com", "google.com.au"} {
		if strings.Contains(w, host) && strings.Contains(cur, host) {
			return true
		}
	}
	// Generic: strip scheme and compare host prefix.
	w = strings.TrimPrefix(w, "https://")
	w = strings.TrimPrefix(w, "http://")
	if i := strings.IndexByte(w, '/'); i >= 0 {
		w = w[:i]
	}
	return w != "" && strings.Contains(cur, w)
}

// ReloadAfterNavigate refreshes the current page after a navigation so first-load
// modals (new collection / project prompts) do not stick over compose.
func ReloadAfterNavigate(page playwright.Page) error {
	if page == nil {
		return fmt.Errorf("no page")
	}
	if _, err := page.Reload(playwright.PageReloadOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60_000),
	}); err != nil {
		return fmt.Errorf("reload after navigate: %w", err)
	}
	return nil
}

// EnsureCompose focuses a fresh compose box (new thread when possible).
func EnsureCompose(page playwright.Page) error {
	if page == nil {
		return fmt.Errorf("no page")
	}
	DismissBlockingDialogs(page)

	// Prefer focusing an already-visible ask box; avoid New Thread unless needed.
	if box, err := composeBox(page); err == nil {
		_ = box.Click(playwright.LocatorClickOptions{
			Force:   playwright.Bool(true),
			Timeout: playwright.Float(5_000),
		})
		DismissBlockingDialogs(page)
		if _, err2 := composeBox(page); err2 == nil {
			return nil
		}
	}

	for _, label := range NewThreadButtonNames {
		loc := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: label})
		n, err := loc.Count()
		if err == nil && n > 0 {
			vis, _ := loc.First().IsVisible()
			if vis {
				if err := loc.First().Click(); err == nil {
					DismissBlockingDialogs(page)
					_ = waitComposeReady(page, 5_000)
					break
				}
			}
		}
	}
	DismissBlockingDialogs(page)
	box, err := composeBox(page)
	if err != nil {
		return &ErrUIChanged{Op: "compose", Msg: err.Error()}
	}
	if err := box.Click(playwright.LocatorClickOptions{
		Force:   playwright.Bool(true),
		Timeout: playwright.Float(8_000),
	}); err != nil {
		return fmt.Errorf("focus compose: %w", err)
	}
	return nil
}

func composeBox(page playwright.Page) (playwright.Locator, error) {
	for _, sel := range ComposeBoxSelectors {
		loc := page.Locator(sel)
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		limit := n
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			cand := loc.Nth(i)
			vis, _ := cand.IsVisible()
			if !vis {
				continue
			}
			if isProjectOrCollectionField(cand) {
				continue
			}
			return cand, nil
		}
	}
	return nil, fmt.Errorf("compose textbox not found")
}

func isProjectOrCollectionField(loc playwright.Locator) bool {
	if loc == nil {
		return false
	}
	aria, _ := loc.GetAttribute("aria-label")
	ph, _ := loc.GetAttribute("placeholder")
	ariaPH, _ := loc.GetAttribute("aria-placeholder")
	blob := strings.TrimSpace(aria + " " + ph + " " + ariaPH)
	return ProjectDialogFieldPattern.MatchString(blob)
}

// DismissBlockingDialogs closes project/collection create overlays that steal focus.
func DismissBlockingDialogs(page playwright.Page) {
	if page == nil {
		return
	}
	_ = page.Keyboard().Press("Escape")
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(1_500)}
	for _, label := range BlockingDialogDismissNames {
		btn := page.GetByRole("button", playwright.PageGetByRoleOptions{
			Name:  regexp.MustCompile("(?i)^" + regexp.QuoteMeta(label) + "$"),
			Exact: playwright.Bool(false),
		})
		n, err := btn.Count()
		if err != nil || n == 0 {
			continue
		}
		vis, _ := btn.First().IsVisible()
		if !vis {
			continue
		}
		_ = btn.First().Click(force)
		_ = page.Keyboard().Press("Escape")
		return
	}
	// Dialog close (X) near project description field.
	dlg := page.Locator(`[role="dialog"], [data-state="open"]`).Filter(playwright.LocatorFilterOptions{
		HasText: ProjectDialogFieldPattern,
	})
	n, _ := dlg.Count()
	if n > 0 {
		closeBtn := dlg.First().GetByRole("button", playwright.LocatorGetByRoleOptions{
			Name: regexp.MustCompile(`(?i)(close|cancel|dismiss)`),
		})
		cn, _ := closeBtn.Count()
		if cn > 0 {
			_ = closeBtn.First().Click(force)
		}
	}
}

// SetMode selects Search or Deep research on the compose controls.
//
// Primary path (current Perplexity UI): focus compose, type "/", pick a modality
// from the command menu (Deep Research is first; confirm with Use when shown).
// Fallback: legacy Search / Deep research pill + menuitem.
func SetMode(page playwright.Page, mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = ModeDeep
	}
	if mode != ModeDeep && mode != ModeSearch {
		return fmt.Errorf("mode must be %q or %q", ModeDeep, ModeSearch)
	}

	dismissCookieBanner(page)
	DismissBlockingDialogs(page)

	// Search is the default compose mode on current Perplexity UI. The "Search"
	// control is often covered by sibling pills (e.g. Computer), so skip toggles
	// when caller asked for search and a compose box is already present.
	if mode == ModeSearch {
		if _, err := composeBox(page); err == nil {
			return nil
		}
	}

	if mode == ModeDeep && deepResearchAlreadyActive(page) {
		return nil
	}

	// Preferred: "/" modality command palette (Deep Research, Model Council, Plan, …).
	if err := setModeViaSlashCommand(page, mode); err == nil {
		return nil
	}

	// Legacy fallback: mode pills near compose.
	if err := setModeViaLegacyPills(page, mode); err == nil {
		return nil
	}

	if mode == ModeSearch {
		return nil // prefer proceed over hard-fail for Search
	}
	return &ErrUIChanged{
		Op:  "mode_open",
		Msg: "could not open modality menu via / or legacy Search/Deep research control",
	}
}

// setModeViaSlashCommand opens Perplexity's modality picker by typing "/" in compose.
func setModeViaSlashCommand(page playwright.Page, mode string) error {
	box, err := composeBox(page)
	if err != nil {
		return err
	}
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(8_000)}
	if err := box.Click(force); err != nil {
		return fmt.Errorf("focus compose for /: %w", err)
	}
	// Clear any residual text so "/" opens the command menu, not a filter inside a draft.
	_ = box.Fill("")
	_ = page.Keyboard().Press("Control+A")
	_ = page.Keyboard().Press("Backspace")

	if err := box.PressSequentially("/", playwright.LocatorPressSequentiallyOptions{
		Delay: playwright.Float(20),
	}); err != nil {
		// Keyboard fallback when contenteditable rejects PressSequentially.
		if err2 := page.Keyboard().Type("/"); err2 != nil {
			return fmt.Errorf("type / for modalities: %w (keyboard: %v)", err, err2)
		}
	}
	_ = waitComposeReady(page, 2_000)

	if mode == ModeDeep {
		if err := pickDeepResearchFromSlashMenu(page); err != nil {
			// Dismiss slash menu so later fallbacks / submit are not blocked.
			_ = page.Keyboard().Press("Escape")
			return err
		}
		_ = waitComposeReady(page, 3_000)
		return nil
	}

	// Search inside slash menu when present; otherwise leave default.
	if err := pickSearchFromSlashMenu(page); err != nil {
		_ = page.Keyboard().Press("Escape")
		// Not fatal: home compose defaults to Search.
		return nil
	}
	_ = waitComposeReady(page, 3_000)
	return nil
}

func pickDeepResearchFromSlashMenu(page playwright.Page) error {
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(8_000)}

	// Filter menu by typing a short label (matches "Deep Research" in the palette).
	_ = page.Keyboard().Type("deep")
	_ = waitComposeReady(page, 1_500)

	item := firstVisibleModalityItem(page, DeepResearchMenuPattern)
	if item == nil {
		return &ErrUIChanged{Op: "mode_select", Msg: "Deep Research not found in / modality menu"}
	}
	if err := item.Click(force); err != nil {
		return &ErrUIChanged{Op: "mode_click", Msg: err.Error()}
	}
	_ = waitComposeReady(page, 1_000)

	// Detail pane "Use" button (screenshot UI) activates the modality.
	if useBtn := firstVisibleUseButton(page); useBtn != nil {
		_ = useBtn.Click(force)
		_ = waitComposeReady(page, 2_000)
	} else {
		// Some builds arm the modality on list selection alone; Enter as backup.
		_ = page.Keyboard().Press("Enter")
		_ = waitComposeReady(page, 1_500)
	}

	// Clear residual "/deep" text if the palette left filter chars in compose.
	if box, err := composeBox(page); err == nil {
		txt, _ := box.InnerText()
		txt = strings.TrimSpace(txt)
		if txt == "" || strings.HasPrefix(txt, "/") || DeepResearchMenuPattern.MatchString(txt) {
			_ = box.Fill("")
		}
	}
	return nil
}

func pickSearchFromSlashMenu(page playwright.Page) error {
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(5_000)}
	item := firstVisibleModalityItem(page, SearchMenuPattern)
	if item == nil {
		return fmt.Errorf("search modality not in / menu")
	}
	if err := item.Click(force); err != nil {
		return err
	}
	if useBtn := firstVisibleUseButton(page); useBtn != nil {
		_ = useBtn.Click(force)
	}
	return nil
}

func firstVisibleModalityItem(page playwright.Page, name *regexp.Regexp) playwright.Locator {
	// Slash palette may expose options, menuitems, buttons, or plain rows.
	// playwright-go wants AriaRole; untyped string constants convert — variables need cast.
	for _, role := range []playwright.AriaRole{"option", "menuitem", "button", "listitem"} {
		loc := page.GetByRole(role, playwright.PageGetByRoleOptions{Name: name})
		if cand := firstVisibleLocator(loc, 12); cand != nil {
			return cand
		}
	}
	// Text fallback (left-rail row label "Deep Research").
	return firstVisibleLocator(page.GetByText(name), 12)
}

func firstVisibleUseButton(page playwright.Page) playwright.Locator {
	loc := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: ModalityUseButtonPattern})
	return firstVisibleLocator(loc, 8)
}

func firstVisibleLocator(loc playwright.Locator, limit int) playwright.Locator {
	if loc == nil {
		return nil
	}
	n, err := loc.Count()
	if err != nil || n == 0 {
		return nil
	}
	if limit <= 0 || n < limit {
		limit = n
	}
	for i := 0; i < limit && i < n; i++ {
		cand := loc.Nth(i)
		vis, _ := cand.IsVisible()
		if vis {
			return cand
		}
	}
	return nil
}

func deepResearchAlreadyActive(page playwright.Page) bool {
	for _, pat := range DeepResearchActivePatterns {
		re := regexp.MustCompile(pat)
		btn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: re})
		if firstVisibleLocator(btn, 6) != nil {
			return true
		}
	}
	return false
}

// setModeViaLegacyPills opens the old Search / Deep research control near compose.
func setModeViaLegacyPills(page playwright.Page, mode string) error {
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(8_000)}

	opened := false
	for _, name := range ModeOpenButtonNames {
		btn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: name, Exact: playwright.Bool(true)})
		n, err := btn.Count()
		if err != nil || n == 0 {
			btn = page.GetByRole("button", playwright.PageGetByRoleOptions{Name: regexp.MustCompile("(?i)^" + regexp.QuoteMeta(name) + "$")})
			n, err = btn.Count()
		}
		if err != nil || n == 0 {
			continue
		}
		if err := btn.First().Click(force); err == nil {
			opened = true
			_ = waitComposeReady(page, 3_000)
			break
		}
	}
	if !opened {
		return fmt.Errorf("legacy mode pills not found")
	}

	if mode == ModeDeep {
		deep := page.GetByRole("menuitem", playwright.PageGetByRoleOptions{Name: DeepResearchMenuPattern})
		n, _ := deep.Count()
		if n == 0 {
			deep = page.GetByText(DeepResearchMenuPattern)
			n, _ = deep.Count()
		}
		if n == 0 {
			return &ErrUIChanged{Op: "mode_select", Msg: "Deep research option not found (legacy menu)"}
		}
		if err := deep.First().Click(force); err != nil {
			return &ErrUIChanged{Op: "mode_click", Msg: err.Error()}
		}
		_ = waitComposeReady(page, 3_000)
		return nil
	}

	searchItem := page.GetByRole("menuitem", playwright.PageGetByRoleOptions{Name: SearchMenuPattern})
	n, _ := searchItem.Count()
	if n > 0 {
		_ = searchItem.First().Click(force)
	}
	_ = waitComposeReady(page, 3_000)
	return nil
}

func dismissCookieBanner(page playwright.Page) {
	for _, label := range CookieDismissButtonNames {
		btn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: regexp.MustCompile("(?i)^" + regexp.QuoteMeta(label) + "$")})
		n, err := btn.Count()
		if err != nil || n == 0 {
			continue
		}
		_ = btn.First().Click(playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(2_000)})
		return
	}
}

// SubmitPrompt types the prompt into compose and submits (Enter).
func SubmitPrompt(page playwright.Page, prompt string) error {
	DismissBlockingDialogs(page)
	box, err := composeBox(page)
	if err != nil {
		return &ErrUIChanged{Op: "compose", Msg: err.Error()}
	}
	if err := box.Click(playwright.LocatorClickOptions{
		Force:   playwright.Bool(true),
		Timeout: playwright.Float(8_000),
	}); err != nil {
		DismissBlockingDialogs(page)
		if err2 := box.Click(playwright.LocatorClickOptions{
			Force:   playwright.Bool(true),
			Timeout: playwright.Float(5_000),
		}); err2 != nil {
			return fmt.Errorf("focus compose: %w", err2)
		}
	}
	if err := box.Fill(""); err != nil {
		_ = err
	}
	typed := false
	if len(prompt) > 120 {
		if err := box.Fill(prompt); err == nil {
			typed = true
		}
	}
	if !typed {
		if err := box.PressSequentially(prompt, playwright.LocatorPressSequentiallyOptions{
			Delay: playwright.Float(1),
		}); err != nil {
			if err2 := box.Fill(prompt); err2 != nil {
				return fmt.Errorf("type prompt: %w (fill: %v)", err, err2)
			}
		}
	}
	if err := box.Press("Enter"); err != nil {
		// Try submit button
		submit := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: SubmitButtonPattern})
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

// ExtractAnswerText returns best-effort latest assistant prose.
func ExtractAnswerText(page playwright.Page) (string, error) {
	if page == nil {
		return "", fmt.Errorf("no page")
	}
	// Prefer main article / prose regions.
	var best string
	for _, sel := range AnswerTextSelectors {
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
	links := page.Locator(CitationLinkSelector)
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
