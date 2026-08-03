---
phase: 11
slug: foundations-schema-origins-tickets-backend
status: draft
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-03
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Regenerated 2026-08-03 (revision iteration 1) against the final plan set (11-01..11-06).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.26.1 `go test` + testify v1.11.1 + testcontainers-go v0.42.0 (postgres:16-alpine) |
| **Config file** | none — every test artifact is created inside the plan task that runs its own verify (no separate Wave 0; scaffolding map below) |
| **Quick run command** | `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -count=1` |
| **Full suite command** | `make test` (`go test -v ./...`) |
| **Estimated runtime** | ~300 seconds |

---

## Sampling Rate

- **After every task commit:** Run the task's `<automated>` verify command from the plan (map below).
- **After every plan wave:** Run `make test` (full suite). NOTE: full-suite green at the end of wave 1 requires 11-01 Task 1 to have fixed the pre-existing red TestMigration011/012 cycle tests — the wave gate runs only after the wave completes, so mid-wave full-suite runs are NOT sampled (dependency_correctness W-2).
- **Before `/gsd-verify-work`:** Full suite must be green.
- **Max feedback latency:** 300 seconds.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | SC-9 (infra) | — | Pre-existing red cycle tests 011/012 fixed via self-seeded pre-state (Pitfall 3); teardown list extended with tickets/ticket_comments/audit_logs | migration cycle (regression) | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration011_ActivityOntology_UpDownUpCycle\|TestMigration012_StaffingSchema_UpDownUpCycle' -count=1` | ✅ existing (modified) | ⬜ pending |
| 11-01-02 | 01 | 1 | TICK-01/FND-01/FND-03/TICK-04/TICK-05 | T-11-11 | Migration 014-017 shapes: tickets kind/status CHECK + dismissed_hours (TICK-04), origins refs-to-type CHECK (FND-01), sold_hours CHECK (FND-03), audit_logs table (TICK-05); all legacy-safe (D-16) | schema check (ls + grep) | `ls migrations/014_ticket_schema.up.sql migrations/014_ticket_schema.down.sql migrations/015_activity_origins.up.sql migrations/015_activity_origins.down.sql migrations/016_contract_sold_hours.up.sql migrations/016_contract_sold_hours.down.sql migrations/017_audit_logs.up.sql migrations/017_audit_logs.down.sql && grep -c 'origin_type IS NULL' migrations/015_activity_origins.up.sql` | ✅ in-plan | ⬜ pending |
| 11-01-03 | 01 | 1 | SC-9 | T-11-11 | Up/down/up cycle tests 014-017; legacy-NULL rows pass new CHECKs; mixed-refs rows fail with constraint errors | migration cycle | `go test ./internal/adapters/secondary/postgres/ -count=1` | ✅ in-plan | ⬜ pending |
| 11-02-01 | 02 | 1 | TICK-01/TICK-02 | — | ADR-P-003 revised: v0.2 lifecycle (7 statuses + reopen + guarded dismissal), closed kind set, preserved hard boundaries | doc grep | `grep -c 'dismissed' "hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md" && grep -c 'kanban' "hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md"` | ✅ existing (revised) | ⬜ pending |
| 11-02-02 | 02 | 1 | FND-01/FND-02 | — | ADR-P-013: origin axis incl. reviewed_by Phase-11 resolution + ErrOriginImmutable + FND-04 fallback | doc grep | `grep -c 'ErrOriginImmutable' "hourglass-vault/decisions/project/ADR-P-013 — Origins.md" && grep -c 'reviewed_by' "hourglass-vault/decisions/project/ADR-P-013 — Origins.md"` | ✅ in-plan | ⬜ pending |
| 11-02-03 | 02 | 1 | FND-04/TICK-05 | T-11-08 | ADR-BE-016: schema encoding + synchronous in-tx audit writes + terminal-activity definition + LoggedHours signature | doc grep | `grep -c 'LoggedHours' "hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md" && grep -c 'in-transaction' "hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md"` | ✅ in-plan | ⬜ pending |
| 11-03-01 | 03 | 1 | FND-02 | T-11-02 | Routing extracted verbatim (BE-014 semantics); time_entry delegates; wiring compiles | build + vet | `go build ./... && go vet ./internal/core/services/routing/ ./internal/core/services/time_entry/` | ✅ in-plan | ⬜ pending |
| 11-03-02 | 03 | 1 | FND-02 | T-11-02 | Routing approver-set semantics pinned (anchored WG, R-2 commercial-without-WG, upward walk, terminal roleGated, skipToFinance) | unit | `go test ./internal/core/services/routing/ ./internal/core/services/time_entry/ -count=1` | ✅ in-plan | ⬜ pending |
| 11-04-01 | 04 | 2 | FND-03 | T-11-03 | validateSoldConfig: support requires period, project forbids period, legacy NULL allowed | unit | `go test ./internal/core/services/contract/ -count=1` | ✅ existing (extended) | ⬜ pending |
| 11-04-02 | 04 | 2 | FND-03 | T-11-03 / T-11-09 | sold_hours round-trip through repo + handler; DB CHECK backstop (contracts_sold_check) | integration | `go test ./internal/adapters/secondary/postgres/ -run TestContract -count=1 && go test ./internal/core/services/contract/ -count=1` | ✅ existing (extended) | ⬜ pending |
| 11-05-01 | 05 | 3 | FND-01/FND-04 | T-11-01 / T-11-10 | Origin refs set once with per-type role gates + same-org validation; update carrying origin fields → ErrOriginImmutable; refs readable from responses | unit + integration | `go test ./internal/core/services/activity/ ./internal/adapters/secondary/postgres/ -run TestActivityOrigin -count=1` | ✅ existing (extended) | ⬜ pending |
| 11-05-02 | 05 | 3 | TICK-01/TICK-05/FND-01 | T-11-08 | Ticket domain (vocabulary + matrix) + general audit repo; ticket Get org-scoped; audit Create synchronous (never fire-and-forget) | unit + integration | `go test ./internal/adapters/secondary/postgres/ -run 'TestTicket' -count=1 && go build ./internal/...` | ✅ in-plan | ⬜ pending |
| 11-05-03 | 05 | 3 | FND-02 | T-11-02 / T-11-08 | ApproveProposal: routing-resolved approver flips is_active + writes synchronous proposal_approved audit row | unit + integration | `go test ./internal/core/services/activity/ -run 'TestProposal\|TestActivityOrigin' -count=1` | ✅ existing (extended) | ⬜ pending |
| 11-06-01 | 06 | 4 | TICK-01/TICK-02/TICK-05 | T-11-04 / T-11-05 / T-11-08 | State machine + D-15 gates; invalid edges → ErrInvalidTransition; resolved blocked on non-terminal; audit rows in same tx | unit + integration | `go test ./internal/core/services/ticket/ -run 'TestTicketCreate\|TestTicketLifecycle' -count=1` | ✅ in-plan | ⬜ pending |
| 11-06-02 | 06 | 4 | TICK-03/TICK-04/TICK-05 | T-11-06 / T-11-07 / T-11-08 | Atomic triage with in-tx plan validation (Pitfall 7); dismissal guard raw Σ; append-only history (no UPDATE/DELETE on audit_logs/ticket_comments) | unit + integration | `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -run 'TestDismissalGuard\|TestTicketTriage\|TestTicketAudit' -count=1` | ✅ in-plan | ⬜ pending |
| 11-06-03 | 06 | 4 | TICK-01..05 | T-11-04..T-11-08 | 9-route ticket API contract: gates, transitions, guards, 409 mappings, no delete/update routes for history/comments | API contract | `go test ./internal/adapters/primary/http/ -run TestTicket -count=1` | ✅ in-plan | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Test Scaffolding (in-plan — no Wave 0)

This plan set has NO separate Wave 0: every artifact the map's commands depend on is created by the task that runs its own verify (the task's `<action>` specifies the files):

| Scaffolding artifact | Built by |
|----------------------|----------|
| 011/012 cycle-test self-seeded pre-state + teardown entries (`tickets`, `ticket_comments`, `audit_logs`) | 11-01 Task 1 |
| Migration cycle tests `TestMigration014..017` (`ontology_extension_migrations_test.go`) | 11-01 Task 3 |
| `internal/core/domain/ticket/` (ticket.go — entity, kind/status constants, transition matrix, sentinels) | 11-05 Task 2 |
| `internal/core/domain/audit/` (audit.go — general `AuditLog` entity) | 11-05 Task 2 |
| `internal/core/ports/ticket_repository.go` + `audit_log_repository.go`; postgres `ticket_repository.go` (Get) + `audit_log_repository.go`; testdata mocks | 11-05 Task 2 |
| `ticket_repository_test.go` (Get org-scope + audit round-trip) | 11-05 Task 2 |
| `ticket_test.go` + `ticket_integration_test.go` (lifecycle/triage/guard/audit coverage) | 11-06 Tasks 1-2 |
| `ticket_handler_test.go` (TestTicketAPI — full API contract) | 11-06 Task 3 |

Executors must create the task's own scaffolding BEFORE running its verify command. Docker/testcontainers availability per RESEARCH.md Environment Availability (integration tests `t.Skip` when Docker absent).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ADR-P-003 revision + ADR-P-013 + ADR-BE-016 recorded in vault decisions folder | SC-9 (ADR part) | Doc artifact, not code | Confirm `hourglass-vault/decisions/project/ADR-P-003` reflects the revised v0.2 lifecycle + preserved hard boundary list; `hourglass-vault/decisions/project/ADR-P-013` exists with the origin-axis record (incl. reviewed_by resolution + FND-04 fallback); `hourglass-vault/decisions/backend/ADR-BE-016` exists with schema encoding + synchronous in-tx audit-write decision + terminal-activity definition |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify — commands reference only artifacts the task itself creates (in-plan scaffolding)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0: N/A — no separate Wave 0; scaffolding is in-plan (11-01 / 11-05 / 11-06)
- [x] No watch-mode flags
- [x] Feedback latency < 300s
- [ ] `nyquist_compliant: true` set in frontmatter — flip after wave 1 completes (requires 11-01 Task 1 green)

**Approval:** pending
