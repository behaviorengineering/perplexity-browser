package perplexity_test

import (
	"testing"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
)

func TestSanitizeTitleHint(t *testing.T) {
	t.Parallel()
	got := perplexity.SanitizeTitleHintForTest("  doctrine\nlookup  ")
	if got != "doctrine lookup" {
		t.Fatalf("got %q", got)
	}
	long := perplexity.SanitizeTitleHintForTest(string(make([]byte, 200)))
	if len(long) > 120 {
		t.Fatalf("len %d", len(long))
	}
}

func TestSelectorRegistryNonEmpty(t *testing.T) {
	t.Parallel()
	if len(perplexity.ComposeBoxSelectors) == 0 || len(perplexity.NewThreadButtonNames) == 0 {
		t.Fatal("selector registry empty")
	}
	if perplexity.ShareButtonPattern == nil || perplexity.DeepResearchMenuPattern == nil {
		t.Fatal("compiled patterns missing")
	}
}
