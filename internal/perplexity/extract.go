package perplexity

import (
	"regexp"
	"strings"
)

// followUpDockBlob matches the follow-up composer chrome as one flattened phrase.
var followUpDockBlob = regexp.MustCompile(`(?is)^(?:ask a follow-up\s*)?(?:search\s+)?(?:computer\s+)?(?:model\s*)?$`)

// IsFollowUpChrome reports whether s is only the follow-up dock (Ask a follow-up /
// Search / Computer / Model), not assistant prose.
func IsFollowUpChrome(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	allNoise := true
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if !noiseLine.MatchString(trim) {
			allNoise = false
			break
		}
	}
	if allNoise {
		return true
	}
	flat := strings.Join(strings.Fields(strings.ToLower(s)), " ")
	return followUpDockBlob.MatchString(flat)
}

// PickAnswerText returns the longest candidate that is not follow-up chrome.
func PickAnswerText(candidates ...string) string {
	var best string
	for _, raw := range candidates {
		text := strings.TrimSpace(raw)
		if text == "" || IsFollowUpChrome(text) {
			continue
		}
		if len(text) > len(best) {
			best = text
		}
	}
	return best
}
