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
	stable  int
	lastLen int
}

func (t *waitTracker) tick(generating bool, answerLen, required int) bool {
	if generating {
		t.stable = 0
		t.lastLen = answerLen
		return false
	}
	if answerLen > 0 && answerLen == t.lastLen {
		t.stable++
	} else {
		t.stable = 0
		t.lastLen = answerLen
	}
	return t.stable >= required
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
	prog := page.GetByText(GeneratingTextPattern)
	pn, _ := prog.Count()
	if pn > 0 {
		vis, _ := prog.First().IsVisible()
		if vis {
			return true, nil
		}
	}
	return false, nil
}
