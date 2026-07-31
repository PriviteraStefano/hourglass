# ADR-BE-005 — Authentication: JWT in HttpOnly Cookies + Refresh Rotation

---
tags: ["adr", "backend", "auth", "security"]

---

# ADR-BE-005 — Authentication: JWT in HttpOnly Cookies + Refresh Rotation

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/auth/`, `internal/cookies/`, `internal/core/services/auth/`, `internal/adapters/primary/http/auth.go`
**Resolves:** audit T5, T6, T9 (P0-5)
**Frontend counterpart:** global knowledge vault `ADR-FE-013` (link, don't restate)

## Context

Hourglass authenticates with two tokens in HttpOnly cookies: a short-lived access JWT and a long-lived refresh token. The audit found three gaps: refresh tokens were **not rotated** on use (theft = access for up to the full 7-day TTL, undetectable); cookie **names diverged** (`cookies.go` defined `access_token` while handlers hard-coded `auth_token`, leaving the helpers dead); and the refresh endpoint carried no additional misuse detection.

## Decision

1. **Two tokens, HttpOnly cookies, never JS-readable.**
   `auth_token`: HS256 JWT, 15 min, claims `UserID, OrganizationID, Role, Email`.
   `refresh_token`: opaque UUID, 7 days, stored server-side **only as SHA-256 hash**.
2. **Rotation on every refresh.** `POST /auth/refresh` validates the presented token, **revokes its hash, issues a new refresh token**, and sets the new cookie. The old token is dead immediately.
3. **Reuse detection.** Presenting a revoked/expired refresh token revokes the **entire token family** for that user (signals theft per NIST refresh-token guidance).
4. **Single source for cookie names.** `internal/cookies` owns `AccessTokenCookieName`/`RefreshTokenCookieName`; handlers use the cookie helpers — no hard-coded names. (`auth_token` is the canonical access name.)
5. **bcrypt cost 12** for password hashing.

## Consequences

* A stolen refresh token yields at most one use before detection; theft is surfaced, not silent.
* Cookie naming is consistent; the helpers in `internal/cookies` are the only path.
* `/auth/refresh` intentionally stays outside `middleware.Auth` (it must work with an expired access token) — its protection is rotation + reuse detection, not the access JWT.
* ⚠️ Client fingerprinting (UA/IP binding) is deferred — rotation is the v0.1 mitigation; fingerprinting is a candidate follow-up.

## Related

* [[ADR-BE-006 — Middleware Composition]] (TryAuth vs Auth)
* [[ADR-BE-012 — Audit Log Writes]] (revocation events should be auditable)
* Global: `ADR-FE-013` (frontend auth contract: `credentials: 'include'`, 401 → refresh → retry)
