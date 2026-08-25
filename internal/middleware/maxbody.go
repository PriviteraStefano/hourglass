package middleware

import (
	"net/http"
	"strings"
)

// MaxBody limits the size of request bodies to protect against memory-exhaustion
// DoS (CONCERNS.md #8). JSON/non-multipart requests larger than maxBytes are
// rejected with 413 before being read; the body is also capped at read time via
// http.MaxBytesReader. Multipart requests (e.g. receipt uploads, which enforce
// their own 10 MB limit) are exempt so they are not double-limited.
func MaxBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > maxBytes {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
