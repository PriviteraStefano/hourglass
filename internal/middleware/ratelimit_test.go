package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
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

	ctx := context.WithValue(context.Background(), UserIDKey, uuid.MustParse("00000000-0000-0000-0000-000000000123"))

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

// TestRateLimit_AnonymousThenAuthenticated_RatchetsLimitUp reproduces the
// "poisoned bucket" bug: a client that starts the window anonymous (low limit)
// and then authenticates mid-window must be judged against the higher
// authenticated limit for the remainder of that window, not the limit that
// was frozen when the first request opened the window.
//
// To exercise the ratchet on a SINGLE shared bucket, both the anonymous and
// authenticated requests are keyed by IP (the authenticated branch in
// getClientKey returns "user:<uuid>", which would be a different bucket, so
// we instead force the same key by not setting a UserID for the "anonymous"
// requests and using a context key that GetUserID does not recognise for
// the "authenticated" ones — but that would also change the key).
//
// The real-world scenario is: the IP-keyed bucket is opened at the anonymous
// limit, then the user authenticates and moves to a user-keyed bucket at the
// auth limit. The ratchet matters when the SAME bucket receives both
// anonymous and authenticated traffic — e.g. when TryAuth fails to validate
// (expired token) on some requests but succeeds on others within the same
// window, and both fall through to the IP key. We simulate that by NOT
// setting UserID on any request (so the key is always "ip:...") but calling
// the limiter with two different effective limits by toggling the context.
//
// Since getClientKey and getLimit both read GetUserID, and we want the same
// key but different limits, we drive allow() directly.
func TestRateLimit_AnonymousThenAuthenticated_RatchetsLimitUp(t *testing.T) {
	limiter := NewRateLimiter(2, 5)
	const key = "ip:192.168.1.1"

	// 1st request opens the window at the anonymous limit (2).
	if !limiter.allow(key, 2) {
		t.Fatal("request 1 (anonymous, limit 2): expected allowed")
	}

	// 2nd request is still anonymous — would be the last allowed at limit 2.
	if !limiter.allow(key, 2) {
		t.Fatal("request 2 (anonymous, limit 2): expected allowed")
	}

	// Now the user authenticates. The bucket must ratchet up to limit 5.
	// Requests 3, 4, 5 should all be allowed (count 3,4,5 <= 5).
	for i := 3; i <= 5; i++ {
		if !limiter.allow(key, 5) {
			t.Errorf("request %d (authenticated, limit 5): expected allowed, bucket was poisoned at anonymous limit 2", i)
		}
	}

	// Request 6 (count=6, limit=5) must now be rejected.
	if limiter.allow(key, 5) {
		t.Error("request 6 (authenticated, limit 5): expected rejected (count 6 > limit 5), got allowed")
	}
}

// TestRateLimit_NoPermanentLimitInflation is a regression test for the
// "permanent limit inflation" bug: a key that briefly presents a higher
// (authenticated) tier must NOT keep that inflated limit after reverting to
// the anonymous tier. Subsequent anonymous requests are limited by the
// anonymous tier, not the previously-seen higher tier.
func TestRateLimit_NoPermanentLimitInflation(t *testing.T) {
	limiter := NewRateLimiter(2, 5)
	const key = "ip:192.168.1.1"

	// 1st request opens the window at the anonymous limit (2).
	if !limiter.allow(key, 2) {
		t.Fatal("request 1 (anonymous, limit 2): expected allowed")
	}

	// A transient higher-tier (authenticated) request arrives.
	if !limiter.allow(key, 5) {
		t.Fatal("request 2 (authenticated, limit 5): expected allowed")
	}

	// Reverting to anonymous: the 3rd request (count=3) must be judged
	// against the anonymous limit (2), not the previously-seen higher tier
	// (5). count 3 > 2 => rejected.
	if limiter.allow(key, 2) {
		t.Error("request 3 (anonymous, limit 2): expected rejected after reverting from higher tier; stored limit must not stay inflated at 5")
	}
}
