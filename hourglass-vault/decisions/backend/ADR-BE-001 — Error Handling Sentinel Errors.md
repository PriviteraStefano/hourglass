# ADR-BE-001 — Error Handling: Sentinel Errors + HTTP Status Mapping

---
tags: ["adr", "backend", "error-handling", "promote"]

---

# ADR-BE-001 — Error Handling: Sentinel Errors + HTTP Status Mapping

**Status:** Accepted · **(promote)** — candidate for the future global Go ADR set
**Date:** 2026-07-28
**Code:** `internal/core/ports/errors.go`, `pkg/api/response.go`, `internal/adapters/secondary/postgres/postgres.go`, all `internal/adapters/primary/http/*.go`
**Resolves:** audit B2, B3 (validation/error paths)

## Context

Errors need a consistent path from a PostgreSQL failure to an HTTP response. Without a convention, each layer invents its own error vocabulary and handlers either leak internals or collapse everything to 500.

## Decision

Four-layer error flow:

1. **Domain / ports — sentinel errors.** Declared as package-level vars (`var ErrNotFound = errors.New("not found")`). Ports define the shared set (`ErrNotFound`, `ErrConflict`, `ErrForeignKey`); services/domains declare specific ones (`ErrInvalidCreds`, `ErrHasActiveProjects`, …).
2. **Adapters — translation at the boundary.** PostgreSQL adapters convert `pgx` errors to port sentinels via `wrapPGError` (no `pgx` types escape the secondary layer).
3. **Handlers — status mapping.** Handlers `switch` on the sentinel to choose the HTTP status (`ErrNotFound→404`, `ErrConflict→409`, `ErrInvalidCreds→401`, default→500). Response body is the `{ "error": "..." }` envelope via `pkg/api.RespondWithError`; success is `{ "data": ... }` via `RespondWithJSON`.
4. **Matching via `errors.Is`** (sentinels may be wrapped).

Conventions:

* Input-decode failures return **400 with a generic message** — the internal JSON error is never leaked (resolves B3-class bugs where decode errors were discarded).
* Validation of identifiers (`uuid.Parse`, etc.) returns an error to the caller — silent zero-value fall-through is forbidden (resolves B2).
* Error wrapping with `%w` is permitted for context; the sentinel must remain `errors.Is`-matchable.

## Consequences

* One error vocabulary across the repo; new features follow the same four layers.
* Internals (SQL, pgx) never reach clients.
* Status codes are decided in exactly one place per error.
* ⚠️ Whether `%w` wrapping becomes *mandatory* (vs. permitted) is left open — formalize when a global Go set is created.

## Related

* [[ADR-BE-012 — Audit Log Writes]], [[ADR-BE-010 — Logging]]
* `internal/core/services/testdata/` mock sentinels must mirror these for tests to be meaningful
