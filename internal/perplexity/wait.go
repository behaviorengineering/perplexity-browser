package perplexity

import (
	"context"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// WaitSettings tunes generation polling.
type WaitSettings struct {
	IdlePollMS  int
	FastPollMS  int
	StablePolls int
	TimeoutMS   int
}

func (s WaitSettings) normalize() WaitSettings {
	out := s
	if out.IdlePollMS <= 0 {
		out.IdlePollMS = 800
	}
	if out.FastPollMS <= 0 {
		out.FastPollMS = 350
	}
	if out.StablePolls <= 0 {
		out.StablePolls = 2
	}
	if out.TimeoutMS <= 0 {
		out.TimeoutMS = 900_000
	}
	return out
}

type waitTracker struct {
	stable     int
	lastLen    int
	idleNotGen int
}

// answerLenNoise is how much related/sources chrome may jitter without resetting stability.
const answerLenNoise = 120

func (t *waitTracker) tick(generating bool, answerLen, required int) bool {
	if generating {
		t.stable = 0
		t.idleNotGen = 0
		t.lastLen = answerLen
		return false
	}
	t.idleNotGen++
	if answerLen <= 0 {
		return false
	}

	delta := answerLen - t.lastLen
	if delta < 0 {
		delta = -delta
	}
	// Treat small post-answer UI churn as stable once we already have a real answer.
	// Do not apply the noise band while the answer is still first growing from a stub.
	noisy := t.lastLen >= 200 && delta > 0 && delta <= answerLenNoise
	if answerLen == t.lastLen || noisy {
		t.stable++
		if answerLen > t.lastLen {
			t.lastLen = answerLen
		}
	} else {
		t.stable = 0
		t.lastLen = answerLen
	}
	if t.stable >= required {
		return true
	}
	// Failsafe: Stop chrome gone for several polls and we already have a real answer.
	// Covers cases where length keeps twitching just above the noise band.
	if t.idleNotGen >= required+4 && answerLen >= 200 {
		return true
	}
	return false
}

// WaitComplete polls until generation finishes or ctx/timeout ends.
// Uses faster polls while generating and slower polls while checking answer stability.
func WaitComplete(ctx context.Context, page playwright.Page, settings WaitSettings) error {
	cfg := settings.normalize()
	deadline := time.Now().Add(time.Duration(cfg.TimeoutMS) * time.Millisecond)
	var tracker waitTracker

	for {
		if err := ctx.Err(); err != nil {
			return context.Canceled
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

		if tracker.tick(generating, cur, cfg.StablePolls) {
			return nil
		}

		interval := pollInterval(generating, cfg.IdlePollMS, cfg.FastPollMS)
		if err := sleepCtx(ctx, interval); err != nil {
			return err
		}
	}
}

func isGenerating(page playwright.Page) (bool, error) {
	if page == nil {
		return false, nil
	}

	// Prefer explicit Stop-generating controls (not bare Cancel / unrelated Stop labels).
	stopBtns := page.GetByRole("button", playwright.PageGetByRoleOptions{
		Name: regexpStopGenerating,
	})
	if n, err := stopBtns.Count(); err == nil && n > 0 {
		limit := n
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			btn := stopBtns.Nth(i)
			vis, _ := btn.IsVisible()
			if !vis {
				continue
			}
			name, _ := btn.GetAttribute("aria-label")
			text, _ := btn.InnerText()
			if looksLikeStopGenerating(name + " " + text) {
				return true, nil
			}
		}
	}

	// Fallback: scan a few visible buttons with tightened matcher (no bare "cancel").
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
		if looksLikeStopGenerating(name + " " + text) {
			return true, nil
		}
	}

	// Status chrome only — never match "thinking/searching" inside the finished answer body.
	for _, sel := range GeneratingChromeSelectors {
		loc := page.Locator(sel)
		cn, err := loc.Count()
		if err != nil || cn == 0 {
			continue
		}
		limit := cn
		if limit > 6 {
			limit = 6
		}
		for i := 0; i < limit; i++ {
			el := loc.Nth(i)
			vis, _ := el.IsVisible()
			if !vis {
				continue
			}
			text, _ := el.InnerText()
			aria, _ := el.GetAttribute("aria-label")
			blob := text + " " + aria
			if GeneratingTextPattern.MatchString(blob) || looksLikeStopGenerating(blob) {
				return true, nil
			}
		}
	}
	return false, nil
}

// looksLikeStopGenerating detects in-flight Stop controls without matching Cancel dialogs.
func looksLikeStopGenerating(blob string) bool {
	b := strings.ToLower(strings.TrimSpace(blob))
	if b == "" {
		return false
	}
	if strings.Contains(b, "account") || strings.Contains(b, "cancel") {
		return false
	}
	if strings.Contains(b, "stop generating") ||
		strings.Contains(b, "stop research") ||
		strings.Contains(b, "stop response") {
		return true
	}
	// Exact-ish Stop button (not "Stopwatch", "desktop", etc.).
	fields := strings.Fields(b)
	for _, f := range fields {
		if f == "stop" {
			return true
		}
	}
	return false
}
