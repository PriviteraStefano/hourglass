package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	anonymousLimit int
	authLimit      int
	requests       map[string]*clientInfo
	mu             sync.RWMutex
}

type clientInfo struct {
	count     int
	windowEnd time.Time
	limit     int
}

func NewRateLimiter(anonymousLimit, authLimit int) *RateLimiter {
	return &RateLimiter{
		anonymousLimit: anonymousLimit,
		authLimit:      authLimit,
		requests:       make(map[string]*clientInfo),
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
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}

	userID, ok := r.Context().Value(UserIDKey).(string)
	if ok && userID != "" {
		return "user:" + userID
	}
	return "ip:" + ip
}

func (rl *RateLimiter) getLimit(r *http.Request) int {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if ok && userID != "" {
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
	return info.count <= info.limit
}
