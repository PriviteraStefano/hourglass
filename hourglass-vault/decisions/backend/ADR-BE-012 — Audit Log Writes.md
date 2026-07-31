# ADR-BE-012 — Audit Log Writes

---
tags: ["adr", "backend", "audit", "reliability"]

---

# ADR-BE-012 — Audit Log Writes

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/core/services/time_entry/time_entry.go`, `internal/core/ports/`, `internal/adapters/secondary/postgres/`
**Resolves:** audit T7

## Context

Time-entry approval actions write audit records via a fire-and-forget goroutine: `go s.auditRepo.Create(ctx, auditLog)`. The goroutine captured the **request context**, which is cancelled when the HTTP response completes — so the write could fail with `context.Canceled`. Errors were silently discarded: no logging, no retry. Audit history (immutable approval trail, a Control-pillar guarantee) could silently lose entries.

## Decision

1. **Detached context.** Background audit writes use `context.Background()` (with a timeout), never the request context.
2. **Errors are logged, never dropped silently.** A failed audit write emits an error log with the entity ID and action; the operation that triggered it still succeeds (audit must not block the workflow), but the failure is visible.
3. **Fire-and-forget is acceptable for v0.1** under (1)+(2). A durable queue/retry is the documented successor if audit volume or compliance needs grow.
4. **Audit records remain immutable** — append-only, no update/delete path (existing Control-pillar rule, restated for completeness).

## Consequences

* Audit entries survive request cancellation; the immutable-history guarantee is real, not best-effort.
* Failures are observable in logs instead of vanishing.
* The trade-off (async, at-least-once-eventually vs. synchronous transactional) is explicit and revisit-able.
* ⚠️ No retry/back-pressure in v0.1 — a DB outage during approval still loses that audit row (logged). Accepted: approvals are the source of truth and can regenerate audit context; durable queue is the follow-up.

## Related

* [[ADR-BE-010 — Logging]] (where failures surface)
* [[ADR-BE-001 — Error Handling]] (error propagation philosophy)
* Control pillar: approval workflow immutability ([[ADR-P-002 — Four Pillars & Feature Purposes]])
