package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit_Anonymous_Returns429AfterLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiter := NewRateLimiter(10, 100)
	middleware := limiter.Middleware(handler)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d after limit, got %d", http.StatusTooManyRequests, rec.Code)
	}
}

func TestRateLimit_DifferentIPs_HaveSeparateLimits(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiter := NewRateLimiter(2, 100)
	middleware := limiter.Middleware(handler)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("IP 1 request %d: expected %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1 after limit: expected %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("IP 2 request %d: expected %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}
}

func TestRateLimit_AuthenticatedUser_HasHigherLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiter := NewRateLimiter(2, 5)
	middleware := limiter.Middleware(handler)

	ctx := context.WithValue(context.Background(), contextKey("userID"), "user-123")

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("auth user request %d: expected %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("auth user after limit: expected %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
}
