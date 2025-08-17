package error

// Option configures an Error during construction via E().
type Option func(*Error)

// defaultHTTPStatus is the default HTTP status used when constructing errors via E and Ensure.
const defaultHTTPStatus = 500

// WithHTTPStatus sets the HTTP status for the error during E() construction.
func WithHTTPStatus(status int) Option { return func(e *Error) { e.httpStatus = status } }

// WithDetail sets the client-safe detail for the error during E() construction.
func WithDetail(detail string) Option { return func(e *Error) { e.detail = detail } }

// WithContext sets the initial context map for the error during E() construction.
// The provided map is defensively cloned.
func WithContext(ctx map[string]any) Option {
	return func(e *Error) { e.context = cloneMap(ctx) }
}

// WithCause sets the underlying cause to be returned by Unwrap().
func WithCause(cause error) Option { return func(e *Error) { e.cause = cause } }

// E is a minimal builder when you don’t want the full New(...) signature.
// Defaults: HTTPStatus=500, Detail="error".
func E(code, key string, opts ...Option) *Error {
	e := &Error{
		httpStatus: defaultHTTPStatus,
		code:       code,
		key:        key,
		detail:     "error",
	}
	for _, o := range opts {
		o(e)
	}

	return e
}
