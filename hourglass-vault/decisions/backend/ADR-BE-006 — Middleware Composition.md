# ADR-BE-006 — Middleware Composition

---
tags: ["adr", "backend", "middleware", "http"]

---

# ADR-BE-006 — Middleware Composition

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/middleware/{middleware,cors,ratelimit,version}.go`, `cmd/server/main.go`

## Context

Cross-cutting concerns (auth, rate limiting, logging, versioning, CORS) are implemented as composable middleware. Order matters: the wrong sequence authenticates before logging, or CORS-wraps after auth so preflight fails.

## Decision

**Pattern:** standard `func(http.Handler) http.Handler` composition.

**Global order (outer → inner), applied once at the mux:**

```
TryAuth → RateLimiter → Logging → APIVersion → CORS → mux
```

* `TryAuth` — *optional* auth: parses a valid JWT into context, never blocks. Lets Logging/version see identity when present.
* `RateLimiter` — global per-IP/per-user limits ([[ADR-BE-007 — Rate Limiting]]).
* `Logging` — method, path, status, duration.
* `APIVersion` — Accept-header version detection ([[ADR-BE-008 — API Versioning Accept Header]]).
* `CORS` — origin whitelist, handles preflight ([[ADR-BE-011 — CORS Policy]]).

**Per-route wrappers** are applied at registration, *inside* the global chain:

* `middleware.Auth(authService, handler)` — required auth; 401 if missing/invalid. Injects `userID, organizationID, role, email` into context.
* `middleware.RequireRole(role, handler)` — role gate on top of `Auth`.
* Route-specific rate limiters wrap the handler directly (e.g. `authRateLimiter.Middleware(HandlerFunc(authHandler.Login))`).

**Context accessors** are the only way handlers read identity: `GetUserID(ctx)`, `GetOrganizationID(ctx)`, `GetRole(ctx)`, `GetEmail(ctx)`. Handlers never parse the JWT themselves.

## Consequences

* One predictable request pipeline; identity is available to inner layers without double-parsing.
* Per-route auth is explicit at the call site in `cmd/server/main.go` — you can see protection by reading the route table.
* ⚠️ `POST /auth/refresh` and `/auth/logout` are deliberately *not* behind `middleware.Auth` — see [[ADR-BE-005 — Auth JWT + Refresh Rotation]] for why.

## Related

* [[ADR-BE-005 — Auth JWT + Refresh Rotation]], [[ADR-BE-007 — Rate Limiting]], [[ADR-BE-008 — API Versioning Accept Header]], [[ADR-BE-011 — CORS Policy]]
