package perplexity

import "regexp"

// UI selector registry — update here when Perplexity changes layout.
// flow.go and export.go should reference these names, not ad-hoc strings.

var (
	// Prefer explicit new-thread labels only. Bare "New" opens Collections/Projects
	// on current Perplexity UI and blocks compose behind a modal.
	NewThreadButtonNames = []string{"New Thread", "New thread"}

	ComposeBoxSelectors = []string{
		`#ask-input`,
		`div[contenteditable="true"][aria-placeholder*="Ask" i]`,
		`div[contenteditable="true"]`,
		`[role="textbox"]`,
	}

	// Dialog / overlay dismiss labels (project / collection create prompts).
	BlockingDialogDismissNames = []string{
		"Cancel", "Close", "Discard", "Not now", "No thanks", "Skip",
	}

	ProjectDialogFieldPattern = regexp.MustCompile(`(?i)(project\s*description|collection|describe what this project)`)

	// Legacy: older UI opened mode via a Search/Deep research pill next to compose.
	ModeOpenButtonNames = []string{"Search", "Deep research", "Research", "Deep Research"}

	// Current UI (2026): type "/" in compose to open the modality command menu.
	// Options include Deep Research, Model Council, Plan mode, Create skill, Settings.
	DeepResearchMenuPattern = regexp.MustCompile(`(?i)deep\s*research`)
	SearchMenuPattern       = regexp.MustCompile(`(?i)^search$`)
	ModalityUseButtonPattern = regexp.MustCompile(`(?i)^use$`)

	// Visible after a modality is active (compose chrome chips / mode label).
	DeepResearchActivePatterns = []string{
		`(?i)^deep\s*research$`,
		`(?i)deep\s*research`,
	}

	CookieDismissButtonNames = []string{"Accept", "Accept all", "I agree", "Got it", "OK"}

	SubmitButtonPattern = regexp.MustCompile(`(?i)(submit|send|ask)`)

	GeneratingTextPattern = regexp.MustCompile(`(?i)(researching|thinking|searching|generating)`)

	// Status chrome only — never scan the whole page (finished answers often contain
	// "thinking" / "searching" and would look "still generating" forever).
	GeneratingChromeSelectors = []string{
		`button[aria-label*="Stop" i]`,
		`button[aria-label*="stop generating" i]`,
		`[aria-live="polite"]`,
		`[aria-live="assertive"]`,
		`[data-testid*="status" i]`,
		`[class*="status" i]`,
	}

	AnswerTextSelectors = []string{
		`[data-testid="answer"]`,
		`main article`,
		`main .prose`,
		`[class*="answer"]`,
		// Do not use bare `main`: related/sources keep mutating length after the answer finishes.
	}

	CitationLinkSelector = `main a[href^="http"]`

	ShareButtonPattern = regexp.MustCompile(`(?i)^share$`)

	// Thread toolbar ⋯ (three dots) next to Share — Export as Markdown lives here, not under Share.
	MoreOptionsNamePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^more\s+options$`),
		regexp.MustCompile(`(?i)^more\s+actions$`),
		regexp.MustCompile(`(?i)^open\s+menu$`),
		regexp.MustCompile(`(?i)^more$`),
		regexp.MustCompile(`(?i)^options$`),
	}
	MoreOptionsAriaSelectors = []string{
		`button[aria-label*="More options" i]`,
		`button[aria-label*="More actions" i]`,
		`button[aria-label*="Open menu" i]`,
		`button[aria-label^="More" i]`,
		`button[aria-haspopup="menu"]`,
	}

	// ⋯ menu items that trigger a markdown download (not clipboard-only).
	ExportMenuPatterns = []string{
		`(?i)export\s+as\s+markdown`,
		`(?i)export.*markdown`,
		`(?i)download.*markdown`,
		`(?i)download\s+as\s+markdown`,
		`(?i)^markdown$`,
		`(?i)export\s+report`,
		`(?i)download\s+report`,
		`(?i)^download$`,
		`(?i)^export$`,
	}

	// Deep Research report chrome (outside ⋯ menu): download / export report controls.
	// Prefer markdown/report labels first so we do not click unrelated Download buttons.
	DeepResearchDownloadPatterns = []string{
		`(?i)export\s+as\s+markdown`,
		`(?i)download.*markdown`,
		`(?i)export\s+report`,
		`(?i)download\s+report`,
		`(?i)download\s+deep\s*research`,
		`(?i)^download$`,
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
