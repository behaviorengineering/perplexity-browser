package perplexity

import (
	"context"
	"fmt"
	"time"

	"github.com/mxschmitt/playwright-go"
)

const (
	minPollMS        = 200
	defaultComposeMS = 8_000
)

// WaitAfterHome waits until compose is ready or hydration timeout (login wall may have no compose).
func WaitAfterHome(page playwright.Page) {
	_ = waitComposeReady(page, defaultComposeMS)
}

// waitComposeReady returns when a compose box is visible or timeout elapses.
func waitComposeReady(page playwright.Page, timeoutMS int) error {
	if page == nil {
		return fmt.Errorf("no page")
	}
	if timeoutMS <= 0 {
		timeoutMS = defaultComposeMS
	}
	deadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	for {
		if _, err := composeBox(page); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("compose not ready within %dms", timeoutMS)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func waitLocatorVisible(loc playwright.Locator, timeoutMS float64) error {
	if loc == nil {
		return fmt.Errorf("no locator")
	}
	n, err := loc.Count()
	if err != nil || n == 0 {
		return fmt.Errorf("locator not found")
	}
	return loc.First().WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeoutMS),
	})
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return context.Canceled
	case <-t.C:
		return nil
	}
}

func pollInterval(generating bool, idlePollMS, fastPollMS int) time.Duration {
	if fastPollMS < minPollMS {
		fastPollMS = minPollMS
	}
	if idlePollMS < fastPollMS {
		idlePollMS = fastPollMS
	}
	if generating {
		return time.Duration(fastPollMS) * time.Millisecond
	}
	return time.Duration(idlePollMS) * time.Millisecond
}
