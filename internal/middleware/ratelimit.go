package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RateLimiter struct {
	anonymousLimit int
	authLimit      int
	requests       map[string]*clientInfo
	mu             sync.RWMutex
	evictStop      chan struct{}
}

type clientInfo struct {
	count     int
	windowEnd time.Time
	limit     int
}

// evictInterval controls how often expired client entries are reclaimed.
const evictInterval = time.Minute

func NewRateLimiter(anonymousLimit, authLimit int) *RateLimiter {
	rl := &RateLimiter{
		anonymousLimit: anonymousLimit,
		authLimit:      authLimit,
		requests:       make(map[string]*clientInfo),
		evictStop:      make(chan struct{}),
	}
	go rl.sweep()
	return rl
}

// sweep periodically removes client entries whose window has expired so the
// requests map cannot grow unbounded (CONCERNS.md #9).
func (rl *RateLimiter) sweep() {
	ticker := time.NewTicker(evictInterval)
	defer ticker.Stop()
	for {
		select {
		case <-rl.evictStop:
			return
		case <-ticker.C:
			rl.evictExpired()
		}
	}
}

func (rl *RateLimiter) evictExpired() {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, info := range rl.requests {
		if now.After(info.windowEnd) {
			delete(rl.requests, k)
		}
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.getClientKey(r)
		limit := rl.getLimit(r)

		if !rl.allow(key, limit) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getClientKey(r *http.Request) string {
	userID := GetUserID(r.Context())
	if userID != (uuid.UUID{}) {
		// Authenticated requests keep their user-scoped key regardless of
		// proxy headers.
		return "user:" + userID.String()
	}

	// Anonymous clients: prefer the first hop of X-Forwarded-For (the
	// original client behind a shared proxy) so that distinct clients behind
	// the same proxy are not collapsed into a single bucket. Fall back to
	// RemoteAddr when no forwarder header is present.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if clientIP := strings.TrimSpace(strings.SplitN(xff, ",", -1)[0]); clientIP != "" {
			return "ip:" + clientIP
		}
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	return "ip:" + ip
}

func (rl *RateLimiter) getLimit(r *http.Request) int {
	userID := GetUserID(r.Context())
	if userID != (uuid.UUID{}) {
		return rl.authLimit
	}
	return rl.anonymousLimit
}

func (rl *RateLimiter) allow(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[key]

	if !exists || now.After(info.windowEnd) {
		rl.requests[key] = &clientInfo{
			count:     1,
			windowEnd: now.Add(time.Minute),
			limit:     limit,
		}
		return true
	}

	info.count++
	// The limit for the current request is the limit of the CURRENT request's
	// tier, recomputed per request. We must NOT permanently inflate the stored
	// limit to the highest tier ever seen in the window: doing so would let a
	// key keep an elevated limit after reverting to a lower tier (e.g. an
	// anonymous client that briefly presented an auth tier). The window's
	// count is cumulative, but each request is judged against its own tier.
	info.limit = limit
	return info.count <= limit
}
