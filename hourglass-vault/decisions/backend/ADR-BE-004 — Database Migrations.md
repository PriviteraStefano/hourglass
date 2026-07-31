# ADR-BE-004 — Database Migrations: Sequential SQL Files

---
tags: ["adr", "backend", "migrations", "postgres"]

---

# ADR-BE-004 — Database Migrations: Sequential SQL Files

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `migrations/`, `cmd/migrate/main.go`, `internal/adapters/secondary/postgres/exported_test_helpers.go`
**Resolves:** audit T3, B1 (schema-level)

## Context

The schema is managed by raw SQL files applied by a bespoke CLI. Two audit findings are schema-level: the time-entry status CHECK constraint lacks the workflow statuses (`pending_manager`, `pending_finance`, `rejected`) — **B1, a P0** — and the `projects` table has duplicate `project_type`/`type` columns with identical CHECK constraints (T3). Numbering has also drifted (000, 003, 004, 005, 006, 008, 009 — 007 missing).

## Decision

* **Format:** `migrations/{NNN}_{name}.up.sql` + `{NNN}_{name}.down.sql`, three-digit sequential numbers. No gaps going forward (renumbering history is not worth it; new files continue from the max).
* **Application:** `go run ./cmd/migrate -up|-down|-all -dir migrations`, applied in sorted order. Migrations are plain SQL — no ORM-generated DDL.
* **Every up has a down.** Down migrations must genuinely reverse the up (drop added columns/constraints), not be stubs.
* **Seeds are separate.** `*_seed.up.sql` files exist but are excluded from test schema setup (`SetupTestSchema` applies non-seed migrations only).
* **Constraint changes are migrations, not code workarounds.** B1 is fixed by a migration extending the status CHECK constraint — the service layer must not mask schema gaps (the `Reject → draft` workaround is exactly the anti-pattern this rule exists to prevent).
* **Schema drift is fixed forward.** T3's duplicate column is resolved by a new migration that consolidates to one column after verifying which the repositories read/write — not by editing `000_full_schema.up.sql` (applied history is immutable).

## Consequences

* Schema history is append-only and reviewable; production and testcontainers share one source of truth.
* B1's fix follows a repeatable pattern (new migration, not a service-layer hack).
* ⚠️ The historical numbering gap (007) is frozen as-is — cosmetic, not worth rewriting applied history.

## Related

* [[ADR-BE-003 — Data Access pgxpool No ORM]], [[ADR-BE-009 — Testing testcontainers testify]]
* `.planning/ROADMAP.md` Phase 6 (approval workflow that B1 blocks)
