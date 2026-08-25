package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery converts a panic in a downstream handler into a clean 500 with a
// generic JSON body, preventing Go's default connection-closing behavior and
// partial/corrupted responses (CONCERNS.md #13).
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered from %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				if w.Header().Get("Content-Type") == "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
