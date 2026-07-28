package perplexity

import "regexp"

// UI selector registry — update here when Perplexity changes layout.
// flow.go and export.go should reference these names, not ad-hoc strings.

var (
	NewThreadButtonNames = []string{"New Thread", "New thread", "New"}

	ComposeBoxSelectors = []string{
		`div[contenteditable="true"]`,
		`[role="textbox"]`,
		`textarea`,
	}

	ModeOpenButtonNames = []string{"Search", "Deep research", "Research"}

	DeepResearchMenuPattern = regexp.MustCompile(`(?i)deep\s*research`)
	SearchMenuPattern       = regexp.MustCompile(`(?i)^search$`)

	CookieDismissButtonNames = []string{"Accept", "Accept all", "I agree", "Got it", "OK"}

	SubmitButtonPattern = regexp.MustCompile(`(?i)(submit|send|ask)`)

	GeneratingTextPattern = regexp.MustCompile(`(?i)(researching|thinking|searching|generating)`)

	AnswerTextSelectors = []string{
		`[data-testid="answer"]`,
		`main article`,
		`main .prose`,
		`[class*="answer"]`,
		`main`,
	}

	CitationLinkSelector = `main a[href^="http"]`

	ShareButtonPattern = regexp.MustCompile(`(?i)^share$`)

	ExportMenuPatterns = []string{
		`(?i)export.*markdown`,
		`(?i)copy as markdown`,
		`(?i)download.*markdown`,
		`(?i)^markdown$`,
		`(?i)export`,
	}

	ThreadTitleRenamePatterns = []string{
		`(?i)^rename`,
		`(?i)rename thread`,
		`(?i)edit title`,
	}

	ThreadTitleInputSelectors = []string{
		`input[placeholder*="title" i]`,
		`[data-testid*="thread-title" i]`,
		`[aria-label*="thread title" i]`,
	}
)
