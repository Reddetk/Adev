package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery returns middleware that recovers from panics and responds with
// a 500 JSON error. When debugMode is true the stack trace is included.
func Recovery(debugMode bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := string(debug.Stack())
					log.Printf("panic recovered: %v\n%s", rec, stack)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					resp := map[string]string{"error": "internal server error"}
					if debugMode {
						resp["stack"] = stack
					}
					_ = json.NewEncoder(w).Encode(resp)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
