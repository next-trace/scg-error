package error_test

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	apiError "github.com/next-trace/scg-error/error"
)

func TestNewAndGetters_ContextIsCloned(t *testing.T) {
	t.Parallel()
	// Empty map input should not leak a non-nil empty map.
	e0 := apiError.New(200, "c", "k", "d", map[string]any{})
	if e0.Context() != nil {
		t.Fatalf("Context for empty map should be nil")
	}

	ctx := map[string]any{"a": 1}
	e := apiError.New(404, "customer.not_found", "not_found", "customer 42 not found", ctx)

	if got, want := e.HTTPStatus(), 404; got != want {
		t.Fatalf("HTTPStatus=%d want=%d", got, want)
	}

	if got, want := e.Code(), "customer.not_found"; got != want {
		t.Fatalf("Code=%q want=%q", got, want)
	}

	if got, want := e.Key(), "not_found"; got != want {
		t.Fatalf("Key=%q want=%q", got, want)
	}

	if got, want := e.Detail(), "customer 42 not found"; got != want {
		t.Fatalf("Detail=%q want=%q", got, want)
	}

	// Modify provided ctx shouldn't affect internal state
	ctx["a"] = 2

	gotCtx := e.Context()
	if want := map[string]any{"a": 1}; !reflect.DeepEqual(gotCtx, want) {
		t.Fatalf("Context()=%v want=%v", gotCtx, want)
	}

	// Mutating returned map must not change internal state
	gotCtx["b"] = 3
	if reflect.DeepEqual(gotCtx, e.Context()) {
		t.Fatalf("Context returned the same map (mutation leaked)")
	}
}

func TestEBuilder_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	e := apiError.E("validation.failed", "validation")
	if e.HTTPStatus() != 500 {
		t.Fatalf("default HTTPStatus=%d want=500", e.HTTPStatus())
	}

	if e.Detail() != "error" {
		t.Fatalf("default Detail=%q want=\"error\"", e.Detail())
	}

	e = apiError.E(
		"validation.failed",
		"validation",
		apiError.WithHTTPStatus(400),
		apiError.WithDetail("payload invalid"),
		apiError.WithContext(map[string]any{"x": 1}),
	)
	if e.HTTPStatus() != 400 {
		t.Fatalf("HTTPStatus=%d want=400", e.HTTPStatus())
	}

	if e.Detail() != "payload invalid" {
		t.Fatalf("Detail=%q", e.Detail())
	}

	if got := e.Context(); !reflect.DeepEqual(got, map[string]any{"x": 1}) {
		t.Fatalf("Context=%v", got)
	}

	// WithContext must clone input map
	m := map[string]any{"k": "v"}
	e = apiError.E("c", "k", apiError.WithContext(m))
	m["k"] = "v2"

	if got := e.Context()["k"]; got != "v" {
		t.Fatalf("Context must be cloned; got=%v", got)
	}
}

func TestContextSetters_SafetyAndOverwrite(t *testing.T) {
	t.Parallel()

	e := apiError.New(500, "x", "y", "z", nil)

	if e.Context() != nil {
		t.Fatalf("expected nil context initially")
	}

	e.WithContextKV("a", 1)

	if got := e.Context()["a"]; got != 1 {
		t.Fatalf("got=%v", got)
	}

	e.WithContextMap(map[string]any{"b": 2, "a": 3}) // overwrite a
	ctx := e.Context()

	if want := map[string]any{"a": 3, "b": 2}; !reflect.DeepEqual(ctx, want) {
		t.Fatalf("ctx=%v want=%v", ctx, want)
	}

	// returned Context must be a fresh clone each time
	ctx["c"] = 9
	if reflect.DeepEqual(ctx, e.Context()) {
		t.Fatalf("Context() must return a fresh clone each call")
	}
}

func TestWrapAndEnsure(t *testing.T) {
	t.Parallel()

	cause := errors.New("row not found")
	e := apiError.Wrap(cause, 404, "customer.not_found", "not_found", "not found", nil)

	if !errors.Is(e, cause) {
		t.Fatalf("wrapped error must match cause with errors.Is")
	}

	var out *apiError.Error
	if !errors.As(e, &out) || out != e {
		t.Fatalf("errors.As should yield *Error itself")
	}

	if got := apiError.Ensure(nil); got != nil {
		t.Fatalf("Ensure(nil) => %v; want nil", got)
	}

	if got := apiError.Ensure(e); got != e {
		t.Fatalf("Ensure(*Error) returned different pointer")
	}

	plain := errors.New("boom")
	wrapped := apiError.Ensure(plain)

	if wrapped == nil {
		t.Fatalf("Ensure(plain) => nil")
	}

	if wrapped.Code() != "internal.error" || wrapped.Key() != "internal" || wrapped.HTTPStatus() != 500 {
		t.Fatalf(
			"Ensure(plain) unexpected fields: code=%s key=%s status=%d",
			wrapped.Code(),
			wrapped.Key(),
			wrapped.HTTPStatus(),
		)
	}

	if !errors.Is(wrapped, plain) {
		t.Fatalf("Ensure must preserve cause for errors.Is")
	}
}

func TestNilReceiverBehaviors(t *testing.T) {
	t.Parallel()

	var e *apiError.Error

	if got := e.Error(); got != "<nil>" {
		t.Fatalf("nil receiver Error()=%q", got)
	}

	if got := e.WithContextKV("a", 1); got != nil {
		t.Fatalf("WithContextKV on nil should return nil receiver")
	}

	if got := e.WithContextMap(nil); got != nil {
		t.Fatalf("WithContextMap on nil should return nil receiver")
	}
}

func TestErrorString_Format(t *testing.T) {
	t.Parallel()

	e := apiError.New(500, "internal.error", "internal", "internal error", map[string]any{"secret": "do-not-leak"})
	msg := e.Error()
	// Must include code, key, status; must not include context or detail.
	if !contains(msg, "internal.error") ||
		!contains(msg, "internal") ||
		!contains(msg, "(500)") {
		t.Fatalf("Error() missing expected parts: %q", msg)
	}

	if contains(msg, "secret") ||
		contains(msg, "do-not-leak") ||
		contains(msg, "internal error") {
		t.Fatalf("Error() leaked detail/context: %q", msg)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// FuzzWithContextKV (no panics, simple expectations).
func FuzzWithContextKV(f *testing.F) {
	f.Add("k", "v")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, k, v string) {
		t.Parallel()

		e := apiError.New(200, "ok", "ok", "ok", nil)
		_ = e.WithContextKV(k, v)
		got := e.Context()

		if k != "" {
			if _, ok := got[k]; !ok {
				t.Fatalf("expected key %q to exist in context", k)
			}
		}
		// Mutations of returned map must not affect internal state
		got[k] = "mut"

		if k != "" && e.Context()[k] == "mut" {
			t.Fatalf("context mutation leaked into internal map")
		}
	})
}

func TestNestedWrap_IsAs(t *testing.T) {
	t.Parallel()

	cause := errors.New("db not found")
	e1 := apiError.Wrap(cause, http.StatusNotFound, "customer.not_found", "not_found", "customer not found", nil)
	e2 := apiError.Wrap(
		e1,
		http.StatusInternalServerError,
		"repository.failure",
		"internal",
		"repository failure",
		map[string]any{"op": "CustomerRepo.Get"},
	)

	if !errors.Is(e2, cause) {
		t.Fatalf("errors.Is(e2, cause) = false; want true")
	}

	var out1 *apiError.Error
	if !errors.As(e2, &out1) {
		t.Fatalf("errors.As(e2, *Error) = false; want true")
	}

	var out2 *apiError.Error
	if !errors.As(e1, &out2) {
		t.Fatalf("errors.As(e1, *Error) = false; want true")
	}
}

func TestNestedWrap_UnwrapAndTopLevel(t *testing.T) {
	t.Parallel()

	cause := errors.New("db not found")
	e1 := apiError.Wrap(cause, http.StatusNotFound, "customer.not_found", "not_found", "customer not found", nil)
	e2 := apiError.Wrap(
		e1,
		http.StatusInternalServerError,
		"repository.failure",
		"internal",
		"repository failure",
		map[string]any{"op": "CustomerRepo.Get"},
	)

	if !errors.Is(e2, e1) {
		t.Fatalf("errors.Is(e2, e1) = false; want true")
	}

	if e2.Code() != "repository.failure" ||
		e2.Key() != "internal" ||
		e2.HTTPStatus() != http.StatusInternalServerError {
		t.Fatalf(
			"top-level fields mismatch for e2: code=%s key=%s status=%d",
			e2.Code(),
			e2.Key(),
			e2.HTTPStatus(),
		)
	}

	if e1.Code() != "customer.not_found" || e1.Key() != "not_found" || e1.HTTPStatus() != http.StatusNotFound {
		t.Fatalf("e1 fields mutated unexpectedly")
	}

	msg := e2.Error()
	if !contains(msg, "repository.failure") ||
		!contains(msg, "internal") ||
		!contains(msg, "(500)") {
		t.Fatalf("e2.Error() missing expected parts: %q", msg)
	}

	if contains(msg, "op") || contains(msg, "CustomerRepo.Get") {
		t.Fatalf("Error() must not include context map: %q", msg)
	}
}

func TestContextImmutabilityDeep(t *testing.T) {
	t.Parallel()

	src := map[string]any{"a": 1, "b": map[string]any{"x": 1}}
	e := apiError.New(http.StatusBadRequest, "validation.failed", "validation", "payload invalid", src)

	c1 := e.Context()
	c1["a"] = 2
	c1["new"] = 9

	if bm, ok := c1["b"].(map[string]any); ok {
		bm["x"] = 2
	}

	c2 := e.Context()

	if av := c2["a"]; av != 1 {
		t.Fatalf("expected a=1 preserved, got=%v", av)
	}

	if _, ok := c2["new"]; ok {
		t.Fatalf("unexpected key 'new' leaked into internal state")
	}

	if bm, ok := c2["b"].(map[string]any); ok {
		if xv := bm["x"]; xv != 1 {
			t.Fatalf("expected nested x=1 preserved, got=%v", xv)
		}
	}

	// Write paths safety
	e = e.WithContextKV("k", 1)
	got := e.Context()
	got["k"] = 99

	if e.Context()["k"].(int) == 99 {
		t.Fatalf("context mutation leaked into internal map via WithContextKV")
	}

	tmp := map[string]any{"p": 1}
	e = e.WithContextMap(tmp)
	tmp["p"] = 2

	if e.Context()["p"].(int) != 1 {
		t.Fatalf("WithContextMap should clone on write for provided map values where necessary")
	}
}

func TestNew_WithCause_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("driver: bad connection")
	e := apiError.New(500, "internal.error", "internal", "internal error", nil, cause)

	if !errors.Is(e, cause) {
		t.Fatalf("errors.Is(e, cause) = false; want true")
	}

	if got := e.Unwrap(); got != cause {
		t.Fatalf("Unwrap() = %v; want %v", got, cause)
	}

	var out *apiError.Error
	if !errors.As(e, &out) || out != e {
		t.Fatalf("errors.As should yield *Error itself")
	}
}

func TestE_WithCauseOption(t *testing.T) {
	t.Parallel()
	cause := errors.New("sql: no rows in result set")
	e := apiError.E("customer.not_found", "not_found",
		apiError.WithHTTPStatus(404),
		apiError.WithDetail("not found"),
		apiError.WithCause(cause),
	)
	if !errors.Is(e, cause) {
		t.Fatalf("errors.Is(e, cause) = false; want true")
	}
	if got := e.Unwrap(); got != cause {
		t.Fatalf("Unwrap() = %v; want %v", got, cause)
	}
}
