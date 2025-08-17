// Package main demonstrates usage of the scg-error package.
package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/next-trace/scg-error/error"
)

func main() {
	// Direct construction
	e := error.New(http.StatusNotFound, "customer.not_found", "not_found", "customer 42 not found", map[string]any{
		"customer_id": "42",
	})
	fmt.Println(e.Error(), e.HTTPStatus(), e.Code(), e.Key(), e.Detail(), e.Context())

	// Wrap unknown cause, add validation issues into Context
	cause := errors.New("row not found")
	err := error.Wrap(cause, http.StatusNotFound, "customer.not_found", "not_found", "customer 42 not found", nil).
		WithContextKV("customer_id", "42").
		WithContextKV("fields", []map[string]any{
			{"field": "id", "rule": "uuid", "message": "must be a valid UUID"},
		})
	fmt.Println(err)
}
