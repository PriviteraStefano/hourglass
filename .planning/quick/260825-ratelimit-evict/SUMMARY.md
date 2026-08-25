---
status: complete
quick_id: 260825-ratelimit-evict
date: 2026-08-25
---

# Summary — 260825-ratelimit-evict

**Concern:** CONCERNS.md #9 Rate-limiter map never evicted — memory leak.

**Change:** `internal/middleware/ratelimit.go` — `RateLimiter` now runs a background sweeper
(`sweep`, every `evictInterval` = 1 minute) that deletes `requests` entries whose `windowEnd` has
passed, guarded by the existing `sync.RWMutex`. `NewRateLimiter` starts the goroutine; an
`evictStop` channel allows clean shutdown.

**Result:** The `requests` map can no longer grow unbounded — every distinct client/IP entry is
reclaimed shortly after its 1-minute window lapses, eliminating the slow memory leak.

**Not addressed:** CONCERNS.md #10 (per-process only) — cross-instance correctness still requires a
shared store (Redis) or gateway-level limiting; out of scope for this fix.

**Verification:** `go build`, `go vet`, `go test ./internal/middleware/...` pass.
