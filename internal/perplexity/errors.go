package perplexity

import (
	"errors"
	"fmt"
)

// Code is a stable machine-readable error category.
type Code string

const (
	CodeInvalid     Code = "invalid"
	CodeUnavailable Code = "unavailable"
)

// Error is a typed perplexity failure.
type Error struct {
	Op   string
	Code Code
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "perplexity: <nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("perplexity.%s: %s: %v", e.Op, e.Msg, e.Err)
	}
	return fmt.Sprintf("perplexity.%s: %s", e.Op, e.Msg)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// CodeOf returns the perplexity code when err wraps *Error.
func CodeOf(err error) Code {
	var pe *Error
	if errors.As(err, &pe) && pe != nil {
		return pe.Code
	}
	return ""
}

func newErr(op string, code Code, msg string, err error) error {
	return &Error{Op: op, Code: code, Msg: msg, Err: err}
}
