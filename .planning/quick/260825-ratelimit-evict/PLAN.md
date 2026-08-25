---
quick_id: 260825-ratelimit-evict
description: Evict expired rate-limiter entries to stop the map memory leak (CONCERNS #9)
date: 2026-08-25
status: complete
---

# Quick Task 260825-ratelimit-evict

## Plan

Address CONCERNS.md #9 "Rate-limiter map never evicted — memory leak".

Task: add a background sweeper to `RateLimiter` that periodically (every minute) removes
`requests` entries whose window has expired, under the existing mutex. `NewRateLimiter` launches
the sweeper goroutine; `evictStop` allows clean shutdown.

> Note: CONCERNS.md #10 (limiter is per-process only) is a separate, infra-level concern
> (needs Redis or gateway enforcement) and is not addressed here — eviction only bounds memory,
> not cross-instance correctness.

## Verify

- `go build ./internal/middleware` passes
- `go vet ./internal/middleware` passes
- `go test ./internal/middleware/...` passes
