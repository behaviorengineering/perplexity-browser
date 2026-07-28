package session

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResolveSessionScope picks the logical session key for thread state on disk.
// Priority: tool session_id → Cursor workspace root (MCP roots/list) → env default.
func (m *Manager) ResolveSessionScope(ctx context.Context, override string, ss *mcp.ServerSession) string {
	if s := strings.TrimSpace(override); s != "" {
		return sanitizeSessionID(s)
	}
	if id := m.scopeFromWorkspaceRoots(ctx, ss); id != "" {
		return id
	}
	return m.resolveSessionID("")
}

func (m *Manager) scopeFromWorkspaceRoots(ctx context.Context, ss *mcp.ServerSession) string {
	if ss == nil {
		return ""
	}
	res, err := ss.ListRoots(ctx, nil)
	if err != nil || res == nil || len(res.Roots) == 0 {
		return ""
	}
	for _, root := range res.Roots {
		if root == nil {
			continue
		}
		if id := scopeFromRootURI(root.URI); id != "" {
			m.log.Info("session scope from workspace root", "scope", id, "root_uri", root.URI, "root_name", root.Name)
			return id
		}
	}
	return ""
}

func scopeFromRootURI(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	path := u.Path
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	base := filepath.Base(path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	return sanitizeSessionID(base)
}
