package middleware

import "net/http"

// contextKey is the private type used for context value keys set by
// middleware in this package (e.g. claims, request ID).
type contextKey string

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain composes multiple middleware functions into a single Middleware.
// Middleware is applied in the order given, so the first argument is the
// outermost wrapper.
func Chain(mws ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}
