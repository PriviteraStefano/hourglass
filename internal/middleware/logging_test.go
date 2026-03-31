package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func TestLoggingMiddleware_LogsMethodPathStatusDuration(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	})

	middleware := Logging(handler)
	req := httptest.NewRequest(http.MethodPost, "/test/path", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "POST") {
		t.Error("expected log to contain method POST")
	}
	if !strings.Contains(logOutput, "/test/path") {
		t.Error("expected log to contain path /test/path")
	}
	if !strings.Contains(logOutput, "201") {
		t.Error("expected log to contain status 201")
	}
	if !strings.Contains(logOutput, "ms") {
		t.Error("expected log to contain duration in ms")
	}
}
