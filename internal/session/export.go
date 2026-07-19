package session

import (
	"context"
	"os"
	"strings"

	"github.com/xynova/perplexity-browser/internal/perplexity"
	"github.com/xynova/perplexity-browser/internal/result"
)

const exportPreviewChars = 8000

// Export writes the current thread conversation to markdown under saveDir
// (or the configured export dir). Prefers UI export; falls back to scrape.
func (m *Manager) Export(ctx context.Context, threadID, format, saveDir string) result.Export {
	_ = ctx
	out := result.Export{
		Base:   result.Base{Busy: true},
		Format: "markdown",
	}
	if format != "" && !strings.EqualFold(format, "markdown") && !strings.EqualFold(format, "text") {
		out.Status = result.StatusError
		out.Message = "format must be markdown or text"
		out.Busy = false
		return out
	}
	if strings.EqualFold(format, "text") {
		out.Format = "text"
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
		out.Message = "thread_id mismatch with active thread"
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

	// Attempt UI export first (best effort).
	if path, err := perplexity.TryUIExportMarkdown(page, saveDir); err == nil && path != "" {
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

	conv, err := perplexity.ScrapeConversation(page, threadID)
	if err != nil {
		out.Status = result.StatusError
		out.Message = err.Error()
		out.Busy = false
		return out
	}
	md := perplexity.FormatMarkdown(conv)
	if out.Format == "text" {
		md = stripMD(md)
	}
	path, err := perplexity.WriteExportFile(saveDir, threadID, md)
	if err != nil {
		out.Status = result.StatusError
		out.Message = err.Error()
		out.Busy = false
		return out
	}
	preview, trunc := previewMarkdown(md, exportPreviewChars)
	out.Status = result.StatusOK
	out.Method = "scrape"
	out.Path = path
	out.TurnCount = conv.Turns
	out.MarkdownPreview = preview
	out.PreviewChars = len(preview)
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

func stripMD(s string) string {
	s = strings.ReplaceAll(s, "# ", "")
	s = strings.ReplaceAll(s, "## ", "")
	s = strings.ReplaceAll(s, "### ", "")
	return s
}
