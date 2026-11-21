package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/behaviorengineering/perplexity-browser/internal/perplexity"
)

// ExtractProbe is a headed-smoke snapshot of prompt vs answer before export assembly.
type ExtractProbe struct {
	URL           string `json:"url"`
	UserRaw       string `json:"user_raw"`
	Assistant     string `json:"assistant"`
	UserAssembled string `json:"user_assembled"`
	Reason        string `json:"reason"`
}

// ProbeExtract reads prompt and answer from the active page without writing an export file.
func (m *Manager) ProbeExtract(ctx context.Context, sessionID string) (ExtractProbe, error) {
	sessionID = m.resolveSessionID(sessionID)
	out := ExtractProbe{Reason: "ok"}
	if _, err := m.activateSession(ctx, sessionID); err != nil {
		return out, err
	}
	m.mu.Lock()
	page := m.page
	threadID := m.threadID
	m.mu.Unlock()
	if page == nil {
		out.Reason = "no_page"
		return out, nil
	}
	out.URL = page.URL()
	user, err := perplexity.ExtractUserPrompt(page)
	if err != nil {
		return out, err
	}
	assistant, err := perplexity.ExtractAnswerText(page)
	if err != nil {
		return out, err
	}
	out.UserRaw = user
	out.Assistant = assistant
	conv, err := perplexity.ConversationFromTexts(out.URL, threadID, user, assistant, nil)
	if err != nil {
		out.Reason = "no_conversation"
		return out, nil
	}
	out.UserAssembled = conv.User
	switch {
	case strings.TrimSpace(user) == "":
		out.Reason = "prompt_miss"
	case conv.User == "" && strings.TrimSpace(user) != "":
		out.Reason = "prompt_prefix_of_answer"
	default:
		out.Reason = "prompt_ok"
	}
	return out, nil
}

// PromptDOMNode is one live-page candidate for a user-turn or heading.
type PromptDOMNode struct {
	Tag    string `json:"tag"`
	TestID string `json:"testid"`
	Role   string `json:"role"`
	Aria   string `json:"aria"`
	Class  string `json:"cls"`
	Text   string `json:"text"`
}

// PromptDOMDump is a headed snapshot of user-turn selectors vs live nodes.
type PromptDOMDump struct {
	URL          string          `json:"url"`
	Title        string          `json:"title"`
	OgTitle      string          `json:"og_title"`
	PromptCounts map[string]int  `json:"prompt_counts"`
	TestIDs      []string        `json:"testids"`
	Userish      []PromptDOMNode `json:"userish"`
	Headings     []PromptDOMNode `json:"headings"`
	QueryRegion  *PromptDOMNode  `json:"query_region,omitempty"`
	QueryChain   []PromptDOMNode `json:"query_chain,omitempty"`
	Reason       string          `json:"reason,omitempty"`
}

const promptDOMScript = `() => {
  const clip = (s, n) => (s || '').trim().slice(0, n);
  const clsOf = (el) => {
    const c = el.className;
    if (typeof c === 'string') return clip(c, 160);
    if (c && c.baseVal) return clip(String(c.baseVal), 160);
    return '';
  };
  const node = (el) => ({
    tag: el.tagName.toLowerCase(),
    testid: el.getAttribute('data-testid') || '',
    role: el.getAttribute('role') || '',
    aria: el.getAttribute('aria-label') || '',
    cls: clsOf(el),
    text: clip(el.innerText || '', 240)
  });
  const count = (s) => document.querySelectorAll(s).length;
  const testids = [...new Set([...document.querySelectorAll('[data-testid]')].map(e => e.getAttribute('data-testid')))]
    .filter(Boolean).sort().slice(0, 120);
  const userish = [...document.querySelectorAll('[data-testid], [role="heading"], h1, h2, [aria-label]')]
    .filter(el => {
      const hay = [
        el.getAttribute('data-testid') || '',
        el.getAttribute('aria-label') || '',
        el.getAttribute('role') || '',
        clsOf(el)
      ].join(' ').toLowerCase();
      return /user|query|prompt|question|thread|message|ask/.test(hay);
    }).slice(0, 40).map(node);
  const headings = [...document.querySelectorAll('h1, h2, h3, [role="heading"]')].slice(0, 20).map(node);
  const og = document.querySelector('meta[property="og:title"]');
  const expand = document.querySelector('[data-testid="toggle-query-expand-button"]');
  let query_region = null;
  const query_chain = [];
  if (expand) {
    const root = expand.closest('header, article, section, [class*="query" i]') || expand.parentElement;
    if (root) {
      query_region = node(root);
      query_region.text = clip(root.innerText || '', 1200);
    }
    let el = expand;
    for (let i = 0; i < 10 && el && el !== document.body; i++) {
      query_chain.push(node(el));
      el = el.parentElement;
    }
  }
  return {
    title: document.title || '',
    og_title: (og && og.getAttribute('content')) || '',
    query_region,
    query_chain,
    prompt_counts: {
      'user-message': count('[data-testid="user-message"]'),
      query: count('[data-testid="query"]'),
      'thread-query': count('[data-testid="thread-query"]'),
      answer: count('[data-testid="answer"]'),
      'main-article': count('main article'),
      'main-prose': count('main .prose')
    },
    testids,
    userish,
    headings
  };
}`

// ProbePromptDOM lists live user-turn candidates on the persisted thread page.
func (m *Manager) ProbePromptDOM(ctx context.Context, sessionID string) (PromptDOMDump, error) {
	sessionID = m.resolveSessionID(sessionID)
	out := PromptDOMDump{Reason: "ok"}
	if _, err := m.activateSession(ctx, sessionID); err != nil {
		return out, err
	}
	m.mu.Lock()
	page := m.page
	m.mu.Unlock()
	if page == nil {
		out.Reason = "no_page"
		return out, nil
	}
	out.URL = page.URL()
	if err := waitThreadPainted(ctx, page); err != nil {
		out.Reason = "still_loading"
	}
	raw, err := page.Evaluate(promptDOMScript)
	if err != nil {
		return out, fmt.Errorf("evaluate prompt DOM: %w", err)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return out, fmt.Errorf("marshal prompt DOM: %w", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("decode prompt DOM: %w", err)
	}
	out.URL = page.URL()
	if out.Reason == "" {
		out.Reason = "ok"
	}
	return out, nil
}

type threadPaintState struct {
	Loading bool `json:"loading"`
	Answer  int  `json:"answer"`
}

func waitThreadPainted(ctx context.Context, page playwright.Page) error {
	if page == nil {
		return fmt.Errorf("wait thread painted: no page")
	}
	deadline := time.Now().Add(45 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("wait thread painted: %w", err)
		}
		raw, err := page.Evaluate(`() => ({
  loading: !!document.querySelector('[aria-label="Loading thread"]'),
  answer: document.querySelectorAll('[data-testid="answer"], main article, main .prose').length
})`)
		if err != nil {
			return fmt.Errorf("wait thread painted: %w", err)
		}
		b, err := json.Marshal(raw)
		if err != nil {
			return fmt.Errorf("wait thread painted: marshal: %w", err)
		}
		var st threadPaintState
		if err := json.Unmarshal(b, &st); err != nil {
			return fmt.Errorf("wait thread painted: decode: %w", err)
		}
		if !st.Loading && st.Answer > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait thread painted: still loading=%t answer=%d", st.Loading, st.Answer)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait thread painted: %w", ctx.Err())
		case <-time.After(400 * time.Millisecond):
		}
	}
}
