# ADR-BE-011 — CORS Policy: Explicit Whitelist, No Wildcard

---
tags: ["adr", "backend", "cors", "security"]

---

# ADR-BE-011 — CORS Policy: Explicit Whitelist, No Wildcard

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/middleware/cors.go`, `cmd/server/main.go`
**Resolves:** audit S2

## Context

The CORS middleware treated `"*"` as a valid entry in the allowed-origins list. Because the app uses cookie auth (`credentials: 'include'`), a wildcard origin would let any website make credentialed requests — a CSRF-adjacent exposure if `ALLOWED_ORIGINS=*` were ever set in production.

## Decision

* **Whitelist only.** Allowed origins come from `ALLOWED_ORIGINS` (comma-separated); each must be an explicit origin. Default `http://localhost:3000`.
* **Wildcard removed.** `"*"` is not accepted as an origin entry — there is no code path that enables credentialed any-origin requests.
* **Credentials allowed** for whitelisted origins (required for the cookie auth flow).
* **Preflight handled** in the middleware (OPTIONS short-circuit).

## Consequences

* Credentialed cross-origin access is impossible except from named origins — the wildcard foot-gun is removed at the code level, not just by config discipline.
* Production misconfiguration surface shrinks (S2 closed).
* ⚠️ Cookie-based auth still relies on SameSite + origin checks together; CORS is one layer, not the whole CSRF story (see [[ADR-BE-005 — Auth JWT + Refresh Rotation]]).

## Related

* [[ADR-BE-006 — Middleware Composition]] (CORS is innermost global layer)
* [[ADR-BE-005 — Auth JWT + Refresh Rotation]]
