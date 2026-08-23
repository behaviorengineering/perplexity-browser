package perplexity

import (
	"regexp"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

const titleHintMaxRunes = 120

var titleNoise = regexp.MustCompile(`\s+`)

// ApplyTitleHint sets the thread title in the Perplexity UI when possible (best effort).
// Call after the thread page exists (post-submit). Never fails the research flow.
func ApplyTitleHint(page playwright.Page, hint string) error {
	hint = sanitizeTitleHint(hint)
	if hint == "" || page == nil {
		return nil
	}
	force := playwright.LocatorClickOptions{Force: playwright.Bool(true), Timeout: playwright.Float(4_000)}

	// Menu path: thread options → Rename.
	for _, pat := range ThreadTitleRenamePatterns {
		re := regexp.MustCompile(pat)
		item := page.GetByRole("menuitem", playwright.PageGetByRoleOptions{Name: re})
		n, _ := item.Count()
		if n == 0 {
			item = page.GetByText(re)
			n, _ = item.Count()
		}
		if n > 0 {
			if err := item.First().Click(force); err == nil {
				time.Sleep(300 * time.Millisecond)
				if filled := fillTitleField(page, hint); filled {
					return nil
				}
			}
		}
	}

	// Direct title inputs.
	if fillTitleField(page, hint) {
		return nil
	}

	// Click a visible thread title header then type (some builds use contenteditable).
	for _, sel := range []string{`h1`, `[data-testid*="title" i]`, `header [contenteditable="true"]`} {
		loc := page.Locator(sel)
		n, _ := loc.Count()
		if n == 0 {
			continue
		}
		first := loc.First()
		vis, _ := first.IsVisible()
		if !vis {
			continue
		}
		if err := first.Click(force); err != nil {
			continue
		}
		time.Sleep(200 * time.Millisecond)
		if fillTitleField(page, hint) {
			return nil
		}
		_ = first.PressSequentially(hint, playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(5)})
		_ = first.Press("Enter")
		return nil
	}
	return nil
}

// SanitizeTitleHintForTest exposes title sanitization for unit tests.
func SanitizeTitleHintForTest(s string) string { return sanitizeTitleHint(s) }

func sanitizeTitleHint(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = titleNoise.ReplaceAllString(s, " ")
	if len(s) > titleHintMaxRunes {
		s = s[:titleHintMaxRunes]
	}
	return strings.TrimSpace(s)
}

func fillTitleField(page playwright.Page, hint string) bool {
	for _, sel := range ThreadTitleInputSelectors {
		loc := page.Locator(sel)
		n, _ := loc.Count()
		if n == 0 {
			continue
		}
		first := loc.First()
		vis, _ := first.IsVisible()
		if !vis {
			continue
		}
		_ = first.Click()
		if err := first.Fill(hint); err != nil {
			_ = first.PressSequentially(hint, playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(5)})
		}
		_ = first.Press("Enter")
		return true
	}
	return false
}
