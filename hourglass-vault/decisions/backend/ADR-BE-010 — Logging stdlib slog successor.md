# ADR-BE-010 — Logging: stdlib log, slog as Successor

---
tags: ["adr", "backend", "logging", "observability"]

---

# ADR-BE-010 — Logging: stdlib log, slog as Successor

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/middleware/middleware.go`, `cmd/server/main.go`

## Context

Logging today is the standard-library `log` package: request logging in middleware (method, path, status, duration ms) and `log.Printf`/`log.Fatal` elsewhere. No structured logger (zerolog/slog), no error-tracking service (Sentry et al.). For v0.1 the question is whether to adopt structure now or accept `log` deliberately.

## Decision

* **v0.1: standard-library `log`, accepted as a deliberate trade-off** — not an oversight. One request line per request from middleware; startup/fatal via `log.Println`/`log.Fatalf`.
* **Log level discipline:** security-relevant failures (refresh-token reuse, audit-write failure, password-reset verify failures) are logged as errors with the entity ID — these are the events [[ADR-BE-012 — Audit Log Writes]] and [[ADR-BE-005 — Auth JWT + Refresh Rotation]] require to be visible.
* **Successor: `log/slog`** (stdlib structured logging) when observability needs grow — chosen now so the migration target isn't re-debated later. No third-party logger.
* **Out of scope for v0.1:** external error tracking (Sentry), metrics, tracing. Deferred, not rejected.

## Consequences

* Zero new dependencies for launch; security events are still greppable.
* The slog migration is a contained refactor (middleware + call sites) with a pre-chosen target.
* ⚠️ Until slog, structured querying (by user, by route) is limited — accepted for a single-instance v0.1.

## Related

* [[ADR-BE-006 — Middleware Composition]] (logging sits in the chain)
* [[ADR-BE-012 — Audit Log Writes]], [[ADR-BE-005 — Auth JWT + Refresh Rotation]] (events that must be logged)
