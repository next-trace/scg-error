# scg-error

Production-grade, transport-agnostic error type and helpers for Go services.

[![CI](https://github.com/next-trace/scg-error/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/next-trace/scg-error/actions/workflows/ci.yml)
[![Coverage ≥ 90% (enforced by CI)](https://img.shields.io/badge/coverage-%E2%89%A5%2090%25-brightgreen)](https://github.com/next-trace/scg-error/actions/workflows/ci.yml)

## Overview
Why this library exists: to provide a small, well-behaved error type that:
- Carries stable machine codes and human-safe detail.
- Stores structured context (map[string]any) without leaking internal state.
- Wraps underlying causes to interoperate with errors.Is / errors.As.
- Stays transport-agnostic; adapters (HTTP/GRPC/etc.) live elsewhere.

Core shape:
- Single concrete type error.Error with fields: HTTPStatus, Code, Key, Detail, Context.
- Minimal contract in contract.Error for other packages to depend on.
- Context maps are defensively cloned on read and write; never leaked.

## Install

```
go get github.com/next-trace/scg-error
```

## Usage

Tip: Code to the interface. Import the concrete implementation for construction, but in your service boundaries (handlers, use-cases), accept and return the stable interface contract.Error. This prevents downstream breakage if the internal implementation evolves.

Create with code and context:

```go
package main

import (
	"fmt"
	"net/http"

	scgerr "github.com/next-trace/scg-error/error"
)

func main() {
	e := scgerr.New(
		http.StatusNotFound,
		"customer.not_found",
		"not_found",
		"customer 42 not found",
		map[string]any{"customer_id": "42"},
	)
	fmt.Println(e.Code(), e.Key(), e.HTTPStatus())
}
```

Builder with options:

```go
package main

import (
	"fmt"

	scgerr "github.com/next-trace/scg-error/error"
)

func main() {
	e := scgerr.E(
		"validation.failed",
		"validation",
		scgerr.WithHTTPStatus(400),
		scgerr.WithDetail("payload invalid"),
		scgerr.WithContext(map[string]any{
			"fields": []map[string]any{{
				"field": "email", "rule": "email", "message": "invalid",
			}},
		}),
	)
	fmt.Println(e.Detail())
}
```

Wrap/unwrap with errors.Is / errors.As:

```go
package main

import (
	"errors"
	"fmt"
	"net/http"

	scgerr "github.com/next-trace/scg-error/error"
)

func main() {
	cause := errors.New("row not found")
	wrap := scgerr.Wrap(cause, http.StatusNotFound, "customer.not_found", "not_found", "not found", nil)

	// errors.Is sees the original cause
	fmt.Println(errors.Is(wrap, cause)) // true

	// errors.As extracts *error.Error
	var e *scgerr.Error
	fmt.Println(errors.As(wrap, &e), e.Code()) // true customer.not_found
}
```

Normalizing any error:

```go
package main

import (
	"errors"
	"fmt"

	scgerr "github.com/next-trace/scg-error/error"
)

func main() {
	var err error = errors.New("boom")
	apiErr := scgerr.Ensure(err)
	fmt.Println(apiErr.Code(), apiErr.HTTPStatus()) // internal.error 500
}
```

Immutability of context (defensive cloning):

```go
package main

import (
	"fmt"
	"net/http"

	scgerr "github.com/next-trace/scg-error/error"
)

func main() {
	src := map[string]any{"a": 1}
	e := scgerr.New(http.StatusBadRequest, "validation.failed", "validation", "payload invalid", src)

	// Changing the source map does not affect the error
	src["a"] = 2
	fmt.Println(e.Context()["a"]) // 1

	// Changing the returned map also does not mutate the error
	c := e.Context()
	c["a"] = 9
	fmt.Println(e.Context()["a"]) // 1
}
```

### Coding to interface

Prefer depending on the stable interface in your app code:

```go
package handler

import (
    "net/http"

    apperr "github.com/next-trace/scg-error/error"
    "github.com/next-trace/scg-error/contract"
)

// Service boundary depends on contract.Error, not the concrete type.
func writeError(w http.ResponseWriter, err contract.Error) {
    // encode based on err.HTTPStatus(), err.Code(), err.Detail(), err.Context()
}

func do() error {
    // Construct concrete error, but return as the built-in error type
    // so callers can use errors.As to get contract.Error or *error.Error.
    e := apperr.New(http.StatusBadRequest, "validation.failed", "validation", "invalid payload", nil)
    return e
}
```

To wrap an underlying cause, either pass it to New as the final variadic argument or use the WithCause option with E:

```go
package main

import (
	"errors"

	apperr "github.com/next-trace/scg-error/error"
)

func main() {
	cause := errors.New("db: bad connection")
	e1 := apperr.New(500, "internal.error", "internal", "internal error", nil, cause)
	e2 := apperr.E("internal.error", "internal", apperr.WithCause(cause))
	_, _ = e1, e2
}
```

### Conventions
- Validation problems: use Context["fields"] as []map[string]any with keys field, rule, message.


## Testing & Quality
- Run tests locally: `go test ./... -race -cover`
- CI runs build, lint, security checks, and enforces coverage ≥ 90%: see the CI badge above.

## Versioning

This project follows [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`).

- **MAJOR**: Breaking API changes
- **MINOR**: New features (backward-compatible)
- **PATCH**: Bug fixes and improvements (backward-compatible)

Consumers should always pin to a specific tag (e.g. `v1.2.3`) to avoid accidental breaking changes.


## License
MIT

## Security
Please report security issues privately via GitHub Security Advisories (Create advisory in the repository’s Security tab). Avoid filing public issues for vulnerabilities.

## Changelog
See GitHub Releases for tagged changes: https://github.com/next-trace/scg-error/releases