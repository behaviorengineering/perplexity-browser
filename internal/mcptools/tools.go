package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
	"github.com/behaviorengineering/perplexity-browser/internal/session"
)

// Register attaches v1 tools to the MCP server.
func Register(server *mcp.Server, mgr *session.Manager, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	h := &handlers{mgr: mgr, log: log}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "perplexity_session",
		Description: "Check login/browser status, wait for human login, cancel an in-flight wait, or close the browser (keeps the profile).",
	}, h.session)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "perplexity_research",
		Description: "Start a new Perplexity thread (Search or Deep research), submit a prepared prompt, and wait for the answer.",
	}, h.research)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "perplexity_continue",
		Description: "Send a follow-up message in the active Perplexity thread and wait for the next answer.",
	}, h.continueTurn)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "perplexity_export",
		Description: "Export the full conversation via Perplexity Share UI (markdown file). Returns export_manual if UI export fails so the human can Share/copy manually.",
	}, h.export)
}

type handlers struct {
	mgr *session.Manager
	log *slog.Logger
}

type sessionIn struct {
	Action    string `json:"action" jsonschema:"status, wait_for_login, close, or cancel"`
	SessionID string `json:"session_id,omitempty" jsonschema:"optional override; default is Cursor workspace folder name from MCP roots"`
	TimeoutMS int    `json:"timeout_ms,omitempty" jsonschema:"timeout for wait_for_login in milliseconds"`
}

type researchIn struct {
	Prompt    string `json:"prompt" jsonschema:"full prepared prompt text"`
	Mode      string `json:"mode,omitempty" jsonschema:"deep or search; default deep"`
	TitleHint string `json:"title_hint,omitempty" jsonschema:"optional thread title hint without case PII"`
	SessionID string `json:"session_id,omitempty" jsonschema:"optional override; default is Cursor workspace folder name from MCP roots"`
	TimeoutMS int    `json:"timeout_ms,omitempty" jsonschema:"override wait timeout in milliseconds"`
}

type continueIn struct {
	Message   string `json:"message" jsonschema:"follow-up message"`
	ThreadID  string `json:"thread_id,omitempty" jsonschema:"optional thread id; defaults to active for session"`
	SessionID string `json:"session_id,omitempty" jsonschema:"optional override; default is Cursor workspace folder name from MCP roots"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
}

type exportIn struct {
	ThreadID  string `json:"thread_id,omitempty"`
	SessionID string `json:"session_id,omitempty" jsonschema:"optional override; default is Cursor workspace folder name from MCP roots"`
	Format    string `json:"format,omitempty" jsonschema:"markdown or text"`
	SaveDir   string `json:"save_dir,omitempty"`
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: `{"status":"error","message":"marshal failed"}`}},
			IsError: true,
		}, nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, v, nil
}

func (h *handlers) session(ctx context.Context, req *mcp.CallToolRequest, in sessionIn) (*mcp.CallToolResult, any, error) {
	action := in.Action
	if action == "" {
		action = "status"
	}
	scope := h.mgr.ResolveSessionScope(ctx, in.SessionID, req.Session)
	switch action {
	case "status":
		s := h.mgr.Status(ctx, true, scope)
		return jsonResult(s)
	case "wait_for_login":
		if !h.mgr.TryBegin() {
			return jsonResult(result.Session{
				Base: result.Base{Status: result.StatusBusy, Message: "another tool is running", Busy: true},
			})
		}
		defer h.mgr.End()
		s := h.mgr.WaitForLogin(ctx, in.TimeoutMS, scope)
		return jsonResult(s)
	case "cancel":
		h.mgr.Cancel()
		return jsonResult(result.Session{
			Base: result.Base{Status: result.StatusOK, Message: "cancel signalled", Busy: h.mgr.IsBusy()},
		})
	case "close":
		if err := h.mgr.CloseBrowser(); err != nil {
			return jsonResult(result.Session{
				Base: result.Base{Status: result.StatusError, Message: err.Error()},
			})
		}
		return jsonResult(result.Session{
			Base:        result.Base{Status: result.StatusOK, Message: "browser closed; profile kept"},
			UserDataDir: h.mgr.Config().UserDataDir,
			ExportDir:   h.mgr.Config().ExportDir,
			BrowserOpen: false,
		})
	default:
		return jsonResult(result.Session{
			Base: result.Base{
				Status:  result.StatusError,
				Message: fmt.Sprintf("unknown action %q (use status, wait_for_login, close, cancel)", action),
			},
		})
	}
}

func (h *handlers) research(ctx context.Context, req *mcp.CallToolRequest, in researchIn) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Prompt) == "" {
		return jsonResult(result.Turn{
			Base: result.Base{Status: result.StatusError, Message: "prompt is required"},
		})
	}
	if !h.mgr.TryBegin() {
		return jsonResult(result.Turn{
			Base: result.Base{Status: result.StatusBusy, Message: "another tool is running", Busy: true},
		})
	}
	defer h.mgr.End()
	scope := h.mgr.ResolveSessionScope(ctx, in.SessionID, req.Session)
	return jsonResult(h.mgr.Research(ctx, in.Prompt, in.Mode, in.TitleHint, scope, in.TimeoutMS))
}

func (h *handlers) continueTurn(ctx context.Context, req *mcp.CallToolRequest, in continueIn) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Message) == "" {
		return jsonResult(result.Turn{
			Base: result.Base{Status: result.StatusError, Message: "message is required"},
		})
	}
	if !h.mgr.TryBegin() {
		return jsonResult(result.Turn{
			Base: result.Base{Status: result.StatusBusy, Message: "another tool is running", Busy: true},
		})
	}
	defer h.mgr.End()
	scope := h.mgr.ResolveSessionScope(ctx, in.SessionID, req.Session)
	return jsonResult(h.mgr.Continue(ctx, in.Message, in.ThreadID, scope, in.TimeoutMS))
}

func (h *handlers) export(ctx context.Context, req *mcp.CallToolRequest, in exportIn) (*mcp.CallToolResult, any, error) {
	if !h.mgr.TryBegin() {
		return jsonResult(result.Export{
			Base: result.Base{Status: result.StatusBusy, Message: "another tool is running", Busy: true},
		})
	}
	defer h.mgr.End()
	scope := h.mgr.ResolveSessionScope(ctx, in.SessionID, req.Session)
	return jsonResult(h.mgr.Export(ctx, in.ThreadID, in.Format, in.SaveDir, scope))
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
