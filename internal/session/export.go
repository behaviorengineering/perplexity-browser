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

const exportManualHint = "UI export failed: in the headed browser use Share → copy or download markdown, then paste or save the file for the assistant"

// Export writes the current thread via Perplexity UI export (Share → markdown).
// On failure returns export_manual so callers ask the human to export from the thread.
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

	path, err := perplexity.TryUIExportMarkdown(page, saveDir)
	if err != nil || path == "" {
		out.Status = result.StatusExportManual
		out.Message = exportManualHint
		if err != nil {
			out.Message = exportManualHint + " (" + err.Error() + ")"
		}
		out.Busy = false
		return out
	}

	body, _ := os.ReadFile(path)
	preview, trunc := previewMarkdown(string(body), exportPreviewChars)
	out.Status = result.StatusOK
	out.Method = "ui_export"
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
