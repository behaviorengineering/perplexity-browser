package session

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

const exportPreviewChars = 8000

const exportManualHint = "UI export failed: in the headed browser use ⋯ → Export as Markdown (or Deep Research Download), then paste or save the file for the assistant"

// Export writes the current thread via Perplexity UI export (⋯ menu / Deep Research download),
// falling back to page scrape when the download UI path fails.
func (m *Manager) Export(ctx context.Context, threadID, format, saveDir, sessionID string) result.Export {
	sessionID = m.resolveSessionID(sessionID)
	out := result.Export{
		Base:   result.Base{Busy: true},
		Format: "markdown",
	}
	m.stampSession(&out.Base, sessionID)

	if format != "" && !strings.EqualFold(format, "markdown") && !strings.EqualFold(format, "text") {
		out.Status = result.StatusError
		out.Message = "format must be markdown or text"
		out.Busy = false
		return out
	}
	if strings.EqualFold(format, "text") {
		out.Format = "text"
	}

	if _, err := m.activateSession(ctx, sessionID); err != nil {
		out.Status = result.StatusError
		out.Message = err.Error()
		out.Busy = false
		return out
	}

	m.mu.Lock()
	page := m.page
	active := m.threadID
	exportDir := m.cfg.ExportDir
	m.mu.Unlock()

	if page == nil {
		out.Status = result.StatusError
		out.Message = "browser not open; run research first or attach CDP Chrome"
		out.Busy = false
		return out
	}
	if threadID != "" && active != "" && threadID != active {
		out.Status = result.StatusError
		out.Message = fmt.Sprintf("thread_id mismatch: session %q has %s, requested %s", sessionID, active, threadID)
		out.ThreadID = active
		out.URL = page.URL()
		out.Busy = false
		return out
	}
	if threadID == "" {
		threadID = active
	}
	if saveDir == "" {
		saveDir = exportDir
	}

	out.ThreadID = threadID
	out.URL = page.URL()

	path, uiErr := perplexity.TryUIExportMarkdown(page, saveDir, threadID)
	if uiErr == nil && path != "" {
		return m.finishExportOK(out, path, "ui_export")
	}

	// Scrape fallback: captures the visible thread answer (often a Deep Research summary).
	// Full report bytes still prefer the UI download path above.
	conv, scrapeErr := perplexity.ScrapeConversation(page, threadID)
	if scrapeErr == nil {
		md := perplexity.FormatMarkdown(conv)
		path, writeErr := perplexity.WriteExportFile(saveDir, threadID, md)
		if writeErr == nil && path != "" {
			out = m.finishExportOK(out, path, "scrape")
			if uiErr != nil {
				out.Message = "ok via scrape fallback (UI download failed: " + uiErr.Error() + ")"
			} else {
				out.Message = "ok via scrape fallback (UI download did not produce a file)"
			}
			return out
		}
		if writeErr != nil {
			uiErr = fmt.Errorf("%v; scrape write: %w", uiErr, writeErr)
		}
	} else if uiErr != nil {
		uiErr = fmt.Errorf("%v; scrape: %w", uiErr, scrapeErr)
	} else {
		uiErr = scrapeErr
	}

	out.Status = result.StatusExportManual
	out.Message = exportManualHint
	if uiErr != nil {
		out.Message = exportManualHint + " (" + uiErr.Error() + ")"
	}
	out.Busy = false
	return out
}

func (m *Manager) finishExportOK(out result.Export, path, method string) result.Export {
	body, _ := os.ReadFile(path)
	preview, trunc := previewMarkdown(string(body), exportPreviewChars)
	out.Status = result.StatusOK
	out.Method = method
	out.Path = path
	out.MarkdownPreview = preview
	out.PreviewChars = len(preview)
	out.TurnCount = countTurns(string(body))
	out.Busy = false
	out.Message = "ok"
	if trunc {
		out.Message = "ok (preview truncated; full file on disk)"
	}
	return out
}

func previewMarkdown(s string, max int) (string, bool) {
	if max <= 0 || len(s) <= max {
		return s, false
	}
	return s[:max] + "\n…[truncated]", true
}

func countTurns(md string) int {
	n := 0
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "## Turn ") {
			n++
		}
	}
	return n
}
