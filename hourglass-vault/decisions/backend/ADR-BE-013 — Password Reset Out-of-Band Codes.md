# ADR-BE-013 — Password Reset: Out-of-Band Codes Only

---
tags: ["adr", "backend", "auth", "security"]

---

# ADR-BE-013 — Password Reset: Out-of-Band Codes Only

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/core/services/password_reset/`, `internal/adapters/primary/http/password_reset.go`
**Resolves:** audit T8 (P0-6)

## Context

`POST /auth/password-reset/request` returned the generated reset code **in the response body** (`"code": code`), and the code was a 3-digit numeric string (~10³ combinations). Anyone able to read the API response could reset the account, and the code was trivially brute-forceable even without it. Rate limiting on the endpoint existed (3/min) but did not compensate for leaking the secret.

## Decision

1. **The reset code is never returned in any API response.** The request endpoint returns only a neutral confirmation (anti-enumeration: same response whether or not the account exists).
2. **Delivery is out-of-band** — email (SMTP integration is scheduled work; until then the code is stored hashed and the delivery seam is abstracted behind the service so the handler never sees the raw code).
3. **Entropy raised**: minimum 6+ alphanumeric characters (cryptographic RNG via `crypto/rand`), replacing the 3-digit numeric code.
4. **Verification is attempt-limited** (in addition to endpoint rate limiting): N failed verifications invalidate the code.
5. **Codes are single-use and short-lived**; successful verify revokes outstanding sessions/tokens for that account.

## Consequences

* The reset secret travels only through the delivery channel, not the API surface.
* Brute-force viability collapses (6+ alphanumeric ≫ 3 digits) and is further capped by attempt limits.
* Neutral responses prevent account enumeration.
* ⚠️ Until SMTP lands, a dev-only path must log the code server-side (never in the response); that path is gated by `GO_ENV != production`.

## Related

* [[ADR-BE-005 — Auth JWT + Refresh Rotation]] (session revocation on reset)
* [[ADR-BE-007 — Rate Limiting]] (endpoint + attempt limits)
* [[ADR-BE-011 — CORS Policy]] (credential surface)
