package pagewatch

import (
	"encoding/json"

	"github.com/mxschmitt/playwright-go"
)

// SessionPort is the CDP session surface used by Watch (fake in tests).
type SessionPort interface {
	Send(method string, params map[string]any) (any, error)
	On(event string, handler func(params map[string]any))
	Detach() error
}

type sessionAPI = SessionPort

func newPlaywrightSession(page PageHook) (sessionAPI, error) {
	if page == nil {
		return nil, newErr("NewSession", CodeInvalid, "nil page", nil)
	}
	pw, ok := page.(playwright.Page)
	if !ok {
		return nil, newErr("NewSession", CodeInvalid, "page is not playwright.Page", nil)
	}
	sess, err := page.Context().NewCDPSession(pw)
	if err != nil {
		return nil, err
	}
	return &playwrightCDPSession{sess: sess}, nil
}

type playwrightCDPSession struct {
	sess playwright.CDPSession
}

func (p *playwrightCDPSession) Send(method string, params map[string]any) (any, error) {
	return p.sess.Send(method, params)
}

func (p *playwrightCDPSession) On(event string, handler func(params map[string]any)) {
	p.sess.On(event, func(params map[string]interface{}) {
		handler(coerceParams(params))
	})
}

func (p *playwrightCDPSession) Detach() error {
	return p.sess.Detach()
}

func coerceParams(in map[string]interface{}) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = coerceValue(v)
	}
	return out
}

func coerceValue(v interface{}) any {
	switch x := v.(type) {
	case map[string]interface{}:
		return coerceParams(x)
	case []interface{}:
		arr := make([]any, len(x))
		for i, el := range x {
			arr[i] = coerceValue(el)
		}
		return arr
	default:
		return v
	}
}

// coerceViaJSON normalizes loosely typed playwright payloads.
func coerceViaJSON(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if m, ok := v.(map[string]interface{}); ok {
		return coerceParams(m)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
