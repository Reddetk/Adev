package middleware

import (
	"log"
	"net/http"
	"time"
)

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the server's own ResponseWriter.
// Without it a handler cannot extend the write deadline, and a slow stage run dies
// with a dropped connection once the server WriteTimeout expires.
func (sw *statusWriter) Unwrap() http.ResponseWriter { return sw.ResponseWriter }

// Logging returns middleware that logs each request with method, path,
// status code, and duration.
// noisyPolls are the endpoints the console polls every few seconds; logging them
// floods the server log with identical lines and hides real work.
var noisyPolls = map[string]bool{
	"/api/v1/operatorrequests": true,
	"/api/v1/modelcalls":       true,
	"/api/v1/activities":       true,
	"/api/v1/logs":             true,
}

func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(sw, r)
			if noisyPolls[r.URL.Path] && r.Method == http.MethodGet {
				return
			}
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.statusCode, time.Since(start))
		})
	}
}
