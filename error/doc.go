// Package error provides a production-grade, transport-agnostic error type and helpers.
//
// It exposes a single concrete type Error that implements contract.Error and integrates
// with the standard library's errors helpers (Is/As) via Unwrap.
//
// Key characteristics:
//   - Stable, machine-facing Code and Key
//   - HTTPStatus for transport adapters
//   - Client-safe Detail
//   - Structured Context map with defensive cloning on read/write
//   - Optional underlying cause preserved for errors.Is / errors.As
//
// Construction options are available via E and With* helpers, and Wrap/Ensure provide
// convenient utilities for adapting arbitrary errors.
package error
