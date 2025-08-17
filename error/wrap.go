package error

import (
	"errors"
)

// Wrap attaches a cause to a new Error. If cause is nil, an opaque cause is created.
// It preserves the original cause for errors.Is / errors.As via Unwrap().
func Wrap(cause error, httpStatus int, code, key, detail string, ctx map[string]any) *Error {
	if cause == nil {
		cause = errors.New("unknown")
	}

	e := New(httpStatus, code, key, detail, ctx)
	e.cause = cause

	return e
}

// Ensure converts any error to *Error.
//
// Behavior:
//   - nil input => nil output
//   - if err is already *Error => returned as-is (same pointer)
//   - otherwise wrap it into a generic internal envelope with HTTP 500 and a safe detail
func Ensure(err error) *Error {
	if err == nil {
		return nil
	}

	var e *Error

	if errors.As(err, &e) {
		return e
	}

	return Wrap(err, defaultHTTPStatus, "internal.error", "internal", "internal error", nil)
}
