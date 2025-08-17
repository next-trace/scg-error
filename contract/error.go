// Package contract exposes the minimal error interface used by other packages.
//
// Implementations must ensure Context returns a defensive copy and support
// errors.Unwrap for proper interoperability with standard error helpers.
package contract

// Error is the minimal, stable surface that other packages can depend on.
//
// Implementations must:
//   - Respect Go initialisms (HTTPStatus).
//   - Ensure Context() returns a defensive copy (never the internal map).
//   - Support errors.Unwrap via Unwrap().
//
// The interface intentionally contains only getters and Unwrap to keep
// the API surface minimal and transport-agnostic.
type Error interface {
	error
	HTTPStatus() int
	Code() string
	Key() string
	Detail() string
	// Context returns a defensive copy; NEVER return the internal map directly.
	Context() map[string]any
	Unwrap() error
}
