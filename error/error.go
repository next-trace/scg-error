// Package error provides a production-grade, transport-agnostic error type and helpers.
//
// It defines a single concrete type Error with immutable, defensively-cloned context
// and support for errors.Is / errors.As via Unwrap.
package error

import (
	"fmt"

	"github.com/next-trace/scg-error/contract"
)

// Error is the canonical error type for SCG services.
//
// Fields:
//   - HTTPStatus: numeric HTTP status (transport-agnostic until encoded)
//   - Code:       stable, machine-facing code (e.g. "customer.not_found")
//   - Key:        category/namespace (e.g. "not_found", "validation")
//   - Detail:     client-safe human detail (no secrets)
//   - Context:    everything else (validation issues, ids, hints, etc.)
type Error struct {
	httpStatus int
	code       string
	key        string
	detail     string
	context    map[string]any
	cause      error
}

// compile-time guarantee that *Error implements contract.Error
var _ contract.Error = (*Error)(nil)

// ------ standard error interface

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	// Compact, dev-friendly string. Clients should read fields/encode via adapters.
	if e.cause != nil {
		return fmt.Sprintf("%s [%s] (%d): %v", e.code, e.key, e.httpStatus, e.cause)
	}

	return fmt.Sprintf("%s [%s] (%d)", e.code, e.key, e.httpStatus)
}

func (e *Error) Unwrap() error { return e.cause }

// ------ contract.Error getters (Go initialisms)

func (e *Error) HTTPStatus() int         { return e.httpStatus }
func (e *Error) Code() string            { return e.code }
func (e *Error) Key() string             { return e.key }
func (e *Error) Detail() string          { return e.detail }
func (e *Error) Context() map[string]any { return cloneMap(e.context) }

// ------ core constructors

// New creates a new Error with the provided fields.
// Context is defensively cloned (pass nil for none).
// The optional cause parameter (if provided) is stored and exposed via Unwrap().
func New(httpStatus int, code, key, detail string, ctx map[string]any, cause ...error) *Error {
	e := &Error{
		httpStatus: httpStatus,
		code:       code,
		key:        key,
		detail:     detail,
		context:    cloneMap(ctx),
	}
	if len(cause) > 0 {
		e.cause = cause[0]
	}
	return e
}

// ------ fluent helpers (chainable, mutate receiver intentionally)

// WithContextKV sets a single key/value in the error context map and returns the same receiver for chaining.
// The internal context map is created on first use.
func (e *Error) WithContextKV(k string, v any) *Error {
	if e == nil {
		return nil
	}

	if e.context == nil {
		e.context = map[string]any{}
	}

	e.context[k] = v

	return e
}

// WithContextMap merges the provided map into the error context and returns the same receiver for chaining.
// Nil or empty maps are ignored. Existing keys are overwritten.
func (e *Error) WithContextMap(m map[string]any) *Error {
	if e == nil || len(m) == 0 {
		return e
	}

	if e.context == nil {
		e.context = map[string]any{}
	}

	for k, v := range m {
		e.context[k] = v
	}

	return e
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]any, len(in))

	for k, v := range in {
		// Deep-clone nested maps with string keys to avoid leaking internal references.
		if mv, ok := v.(map[string]any); ok {
			out[k] = cloneMap(mv)
			continue
		}

		out[k] = v
	}

	return out
}
