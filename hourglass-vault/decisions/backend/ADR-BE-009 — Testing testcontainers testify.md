# ADR-BE-009 — Testing: testcontainers + testify + Handler Integration

---
tags: ["adr", "backend", "testing", "promote"]

---

# ADR-BE-009 — Testing: testcontainers + testify + Handler Integration

**Status:** Accepted · **(promote)** — candidate for the future global Go ADR set
**Date:** 2026-07-28
**Code:** `internal/core/services/testdata/`, `internal/adapters/secondary/postgres/exported_test_helpers.go`, all `*_test.go`, `cmd/server/main_test.go`
**Resolves:** audit B4, §testing-gaps (backend side)

## Context

Phase 0 rebooted testing on PostgreSQL: service tests run against hand-written mocks, and repository/handler tests run against a real database via testcontainers-go. The audit noted the mock layer has correctness holes (e.g. `MockOrgRepo.GetMembership` always returns nil — B4) and that no real-DB integration tests existed before the reboot.

## Decision

Three test tiers, each with a defined truth source:

1. **Service unit tests — mocks.** Table-driven, `testify` `assert`/`require`, mocks from `internal/core/services/testdata/` injected into services. Fast, no DB. `assert.ErrorIs` for sentinel checks.
2. **Repository + handler integration tests — real PostgreSQL via testcontainers.** `TestPool(t)` spins an isolated container; `SetupTestSchema` applies non-seed migrations; `TeardownTestSchema` drops tables. Handler tests wire a real mux on `httptest.NewServer` through the full middleware chain.
3. **E2E smoke — `cmd/server/main_test.go`.** Full server against real PG: health, register, login, authenticated call.

Conventions:

* **Seed factories** (`seedOrg`, `seedUser`, `seedProject`, …) live in `exported_test_helpers.go`, call `t.Helper()`, use `uuid.New()`.
* **Mocks must mirror real sentinel behavior.** Every mock method that a real repository implements with an error path must be able to reproduce it — unconditional `(nil, nil)` returns are bugs (resolves B4: `GetMembership`, `GetDescendants`, `ListMembers` etc. must be settable, not hard-coded nil).
* **No SurrealDB-era helpers** (`GetTestDBWithNamespace`) — dead, replaced by testcontainers.

## Consequences

* SQL errors, constraint violations, and repository bugs are caught in CI, not at runtime (the audit's "no real-DB tests" gap is closed).
* Mock fidelity is enforceable: a test that can't reproduce an error path is a mock bug, filed as such.
* ⚠️ Frontend component-level tests remain a separate, open gap (ADR-FE-011 territory, not this ADR).

## Related

* [[ADR-BE-002 — Hexagonal Wiring]] (why services test clean without a DB)
* [[ADR-BE-003 — Data Access pgxpool No ORM]], [[ADR-BE-004 — Database Migrations]] (test schema source)
* `.planning/codebase/TESTING.md`
