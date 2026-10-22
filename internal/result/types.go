// Package result holds shared MCP tool response shapes.
package result

// Status values for tool JSON bodies.
const (
	StatusOK           = "ok"
	StatusNeedLogin    = "need_login"
	StatusTimeout      = "timeout"
	StatusUIChanged    = "ui_changed"
	StatusBusy         = "busy"
	StatusError        = "error"
	StatusCancelled    = "cancelled"
	StatusNotReady     = "not_ready"
	StatusExportManual = "export_manual" // UI export failed; human Share/copy required
)

// PageEvent is one observe-only page telemetry record.
type PageEvent struct {
	AtMS      int64  `json:"at_ms"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason,omitempty"`
	URL       string `json:"url,omitempty"`
	FromURL   string `json:"from_url,omitempty"`
	Initiator string `json:"initiator,omitempty"`
	Frame     string `json:"frame,omitempty"`
	Message   string `json:"message,omitempty"`
}

// PageWatch is a snapshot of navigation and error observation for one turn.
type PageWatch struct {
	StartedURL      string      `json:"started_url"`
	EndedURL        string      `json:"ended_url"`
	Backend         string      `json:"backend"`
	Events          []PageEvent `json:"events,omitempty"`
	EventsTruncated bool        `json:"events_truncated,omitempty"`
	ResetDetected   bool        `json:"reset_detected"`
	ErrorDetected   bool        `json:"error_detected"`
}

// Citation is a best-effort link from a Perplexity answer.
type Citation struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Base is common fields on every tool response.
type Base struct {
	Status          string `json:"status"`
	Message         string `json:"message,omitempty"`
	ThreadID        string `json:"thread_id,omitempty"`
	URL             string `json:"url,omitempty"`
	Busy            bool   `json:"busy"`
	SessionID       string `json:"session_id,omitempty"`
	ActiveSessionID string `json:"active_session_id,omitempty"`
}

// Turn is a research or continue response.
type Turn struct {
	Base
	Mode            string     `json:"mode,omitempty"`
	Answer          string     `json:"answer,omitempty"`
	AnswerTruncated bool       `json:"answer_truncated,omitempty"`
	AnswerPath      string     `json:"answer_path,omitempty"`
	Citations       []Citation `json:"citations,omitempty"`
	ElapsedMS       int64      `json:"elapsed_ms,omitempty"`
	PageWatch       *PageWatch `json:"page_watch,omitempty"`
}

// Export is a full-conversation export response.
type Export struct {
	Base
	Path            string     `json:"path,omitempty"`
	Format          string     `json:"format,omitempty"`
	Method          string     `json:"method,omitempty"`
	TurnCount       int        `json:"turn_count,omitempty"`
	MarkdownPreview string     `json:"markdown_preview,omitempty"`
	PreviewChars    int        `json:"preview_chars,omitempty"`
	PageWatch       *PageWatch `json:"page_watch,omitempty"`
}

// Session is a perplexity_session response.
type Session struct {
	Base
	LoggedIn    bool       `json:"logged_in"`
	UserDataDir string     `json:"user_data_dir,omitempty"`
	ExportDir   string     `json:"export_dir,omitempty"`
	SessionsDir string     `json:"sessions_dir,omitempty"`
	BrowserOpen bool       `json:"browser_open"`
	PageWatch   *PageWatch `json:"page_watch,omitempty"`
}
