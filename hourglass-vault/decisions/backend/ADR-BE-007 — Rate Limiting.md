# ADR-BE-007 — Rate Limiting: Per-Tier In-Memory Sliding Window

---
tags: ["adr", "backend", "rate-limiting", "security"]

---

# ADR-BE-007 — Rate Limiting: Per-Tier In-Memory Sliding Window

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/middleware/ratelimit.go`, `cmd/server/main.go`
**Resolves:** audit S4, T4 (partially)

## Context

The audit flagged that auth endpoints (login, password reset) shared the global rate limit — no differentiation for credential-sensitive routes — and that the in-memory limiter resets on restart and can't span instances.

## Decision

**Tiered in-memory sliding-window limiters**, all using the same `RateLimiter` primitive (`map[key]*clientInfo` under `sync.RWMutex`, 1-minute window):

| Tier | Limit | Scope |
|------|-------|-------|
| Global anonymous | 20 req/min per IP | all routes (outermost) |
| Global authenticated | 100 req/min per user | all routes |
| Auth (login, register) | 5 req/min | wraps those handlers |
| Password reset | 3 req/min | wraps those handlers |

* Authenticated requests key on user ID; anonymous on IP.
* The auth-tier limit is configurable via `RATE_LIMIT` env (default 5).
* **Accepted limitation (T4):** in-memory state resets on restart and is per-instance. For the single-instance v0.1 deployment this is adequate; a Redis-backed limiter is the documented successor for multi-instance, and is *not* a v0.1 blocker.
* **Map hygiene:** expired-window entries must be evicted (cleanup goroutine or on-access sweep) so the map doesn't grow unbounded with unique IPs (audit "unbounded map" note).

## Consequences

* Credential endpoints get meaningfully stricter limits without a second mechanism.
* The trade-off (simplicity now vs. distributed correctness later) is explicit and deferred behind the same interface.
* ⚠️ Brute-force protection for password reset is *defense in depth* with attempt limits + entropy — rate limiting alone is not the control ([[ADR-BE-013 — Password Reset Out-of-Band Codes]]).

## Related

* [[ADR-BE-006 — Middleware Composition]] (where limiters sit)
* [[ADR-BE-005 — Auth JWT + Refresh Rotation]], [[ADR-BE-013 — Password Reset Out-of-Band Codes]]
