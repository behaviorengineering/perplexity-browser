package pagewatch

import (
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/behaviorengineering/perplexity-browser/internal/result"
)

const maxMessageRunes = 300

// NormalizeURL trims, drops fragment, and returns scheme+host+path+query.
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.Fragment = ""
	if u.Scheme == "" && u.Host == "" {
		return strings.TrimSuffix(raw, "#")
	}
	out := u.String()
	if u.Scheme != "" && u.Host != "" {
		return out
	}
	return strings.TrimSpace(out)
}

// IsComposeResetURL reports whether u looks like the generic compose screen.
func IsComposeResetURL(u string) bool {
	n := NormalizeURL(u)
	if n == "" {
		return true
	}
	low := strings.ToLower(n)
	if low == "about:blank" || strings.HasPrefix(low, "chrome-error://") || strings.HasPrefix(low, "chrome://") {
		return true
	}
	pu, err := url.Parse(n)
	if err != nil || pu.Host == "" {
		return false
	}
	host := strings.ToLower(pu.Host)
	if host != "perplexity.ai" && host != "www.perplexity.ai" {
		return false
	}
	id, ok := searchID(pu)
	if !ok {
		// Home and unknown non-search paths: only "/" is compose.
		path := strings.Trim(pu.Path, "/")
		return path == ""
	}
	// /search, /search/new, and /search/new/<client-uuid> are compose, not a thread.
	return id == "" || id == "new"
}

// IsInThreadURL reports whether u is a Perplexity thread URL (/search/<id>, id not "new").
func IsInThreadURL(u string) bool {
	n := NormalizeURL(u)
	if n == "" {
		return false
	}
	pu, err := url.Parse(n)
	if err != nil {
		return false
	}
	id, ok := searchID(pu)
	if !ok {
		return false
	}
	return id != "" && id != "new"
}

// searchID returns the first /search/ path segment. ok is false when the URL is
// not a perplexity.ai /search route. id is empty for bare /search.
func searchID(pu *url.URL) (id string, ok bool) {
	if pu == nil {
		return "", false
	}
	host := strings.ToLower(pu.Host)
	if host != "perplexity.ai" && host != "www.perplexity.ai" {
		return "", false
	}
	parts := strings.Split(strings.Trim(pu.Path, "/"), "/")
	if len(parts) == 0 || parts[0] != "search" {
		return "", false
	}
	if len(parts) == 1 {
		return "", true
	}
	return parts[1], true
}

// ResetDetected applies compose-reset heuristics from URLs and navigation events.
func ResetDetected(started, ended string, events []result.PageEvent) bool {
	if IsInThreadURL(ended) {
		return false
	}
	for _, ev := range events {
		switch ev.Kind {
		case "navigation", "reload", "script":
			if IsComposeResetURL(ev.URL) {
				return true
			}
		}
	}
	if IsComposeResetURL(ended) && IsInThreadURL(started) {
		return true
	}
	if IsComposeResetURL(ended) && IsComposeResetURL(started) {
		return true
	}
	if IsComposeResetURL(ended) {
		for _, ev := range events {
			if ev.Kind == "reload" || ev.Kind == "script" {
				return true
			}
		}
	}
	return false
}

// ErrorDetected reports page errors, crashes, or failed document loads.
func ErrorDetected(events []result.PageEvent) bool {
	for _, ev := range events {
		switch ev.Kind {
		case "pageerror", "crash", "requestfailed":
			return true
		}
	}
	return false
}

// ClassifyCDP maps a CDP event name and params to a PageEvent when applicable.
func ClassifyCDP(eventName string, params map[string]any, fromURL string, atMS int64) (result.PageEvent, bool) {
	if params == nil {
		return result.PageEvent{}, false
	}
	switch eventName {
	case "Page.frameRequestedNavigation":
		return classifyFrameRequestedNavigation(params, fromURL, atMS)
	case "Page.frameNavigated":
		return classifyFrameNavigated(params, fromURL, atMS)
	case "Runtime.exceptionThrown":
		return classifyExceptionThrown(params, fromURL, atMS)
	case "Network.loadingFailed":
		return classifyLoadingFailed(params, fromURL, atMS)
	case "Network.requestWillBeSent":
		return result.PageEvent{}, false
	default:
		return result.PageEvent{}, false
	}
}

func classifyFrameRequestedNavigation(params map[string]any, fromURL string, atMS int64) (result.PageEvent, bool) {
	if isChildFrame(params) {
		return result.PageEvent{}, false
	}
	reason, _ := stringParam(params, "reason")
	targetURL, _ := stringParam(params, "url")
	kind := "navigation"
	switch reason {
	case "reload":
		kind = "reload"
	case "scriptInitiated":
		kind = "script"
	}
	return result.PageEvent{
		AtMS:    atMS,
		Kind:    kind,
		Reason:  reason,
		URL:     targetURL,
		FromURL: fromURL,
		Frame:   "main",
	}, true
}

func classifyFrameNavigated(params map[string]any, fromURL string, atMS int64) (result.PageEvent, bool) {
	frame, _ := mapParam(params, "frame")
	if frame == nil {
		return result.PageEvent{}, false
	}
	if parentID, ok := stringParam(frame, "parentId"); ok && parentID != "" {
		return result.PageEvent{}, false
	}
	targetURL, _ := stringParam(frame, "url")
	return result.PageEvent{
		AtMS:    atMS,
		Kind:    "navigation",
		URL:     targetURL,
		FromURL: fromURL,
		Frame:   "main",
	}, true
}

func classifyExceptionThrown(params map[string]any, fromURL string, atMS int64) (result.PageEvent, bool) {
	details, _ := mapParam(params, "exceptionDetails")
	msg := ""
	if details != nil {
		if exc, ok := mapParam(details, "exception"); ok && exc != nil {
			msg, _ = stringParam(exc, "description")
		}
		if msg == "" {
			msg, _ = stringParam(details, "text")
		}
	}
	return result.PageEvent{
		AtMS:    atMS,
		Kind:    "pageerror",
		URL:     fromURL,
		FromURL: fromURL,
		Message: truncateRunes(msg, maxMessageRunes),
		Frame:   "main",
	}, true
}

func classifyLoadingFailed(params map[string]any, fromURL string, atMS int64) (result.PageEvent, bool) {
	typ, _ := stringParam(params, "type")
	if typ != "" && !strings.EqualFold(typ, "Document") {
		return result.PageEvent{}, false
	}
	canceled, _ := boolParam(params, "canceled")
	if canceled {
		return result.PageEvent{}, false
	}
	errText, _ := stringParam(params, "errorText")
	targetURL := fromURL
	if req, ok := mapParam(params, "request"); ok && req != nil {
		if u, ok := stringParam(req, "url"); ok && u != "" {
			targetURL = u
		}
	}
	return result.PageEvent{
		AtMS:    atMS,
		Kind:    "requestfailed",
		URL:     targetURL,
		FromURL: fromURL,
		Message: truncateRunes(errText, maxMessageRunes),
		Frame:   "main",
	}, true
}

func isChildFrame(params map[string]any) bool {
	frameID, hasFrame := stringParam(params, "frameId")
	parentID, hasParent := stringParam(params, "parentFrameId")
	if hasParent && parentID != "" {
		return true
	}
	if frame, ok := mapParam(params, "frame"); ok && frame != nil {
		if pid, ok := stringParam(frame, "parentId"); ok && pid != "" {
			return true
		}
	}
	// frameRequestedNavigation uses frameId without parent on main; child often has parentFrameId.
	_ = frameID
	_ = hasFrame
	return false
}

func stringParam(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	switch s := v.(type) {
	case string:
		return s, true
	default:
		return "", false
	}
}

func boolParam(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false, false
	}
	switch b := v.(type) {
	case bool:
		return b, true
	default:
		return false, false
	}
}

func mapParam(m map[string]any, key string) (map[string]any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil, false
	}
	if mm, ok := v.(map[string]any); ok {
		return mm, true
	}
	return nil, false
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var n int
	for i := range s {
		if n >= max {
			return s[:i] + "…"
		}
		n++
	}
	return s
}

// CoalesceWindow is the dedupe window for navigation-family events.
const CoalesceWindow = 500 * time.Millisecond

// KindFamily groups navigation-like kinds for coalescing.
func KindFamily(kind string) string {
	switch kind {
	case "reload", "script", "navigation":
		return "nav"
	default:
		return kind
	}
}
