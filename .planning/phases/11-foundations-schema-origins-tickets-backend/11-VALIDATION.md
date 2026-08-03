---
phase: 11
slug: foundations-schema-origins-tickets-backend
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-03
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.26.1 `go test` + testify v1.11.1 + testcontainers-go v0.42.0 (postgres:16-alpine) |
| **Config file** | none — Wave 0 (patterns from `.planning/codebase/TESTING.md`) |
| **Quick run command** | `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -count=1` |
| **Full suite command** | `make test` (`go test -v ./...`) |
| **Estimated runtime** | ~300 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/core/services/<affected>/ ./internal/adapters/secondary/postgres/ -count=1`
- **After every plan wave:** Run `make test` (full suite)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 300 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | FND-01 | T-11-01 / — | Origin refs set once, immutable, CHECK rejects mixed refs | unit + integration | `go test ./internal/core/services/activity/ ./internal/adapters/secondary/postgres/ -run TestActivityOrigin -count=1` | ❌ W0 | ⬜ pending |
| 11-01-02 | 01 | 1 | FND-02 | T-11-02 / — | Proposal is_active=false, routing flips + audit row, refs never mutated | unit + integration | `go test ./internal/core/services/ticket/ ./internal/core/services/activity/ -run TestProposal -count=1` | ❌ W0 | ⬜ pending |
| 11-02-01 | 02 | 1 | FND-03 | T-11-03 / — | sold_hours read/write; support requires period; project forbids period | unit + integration | `go test ./internal/core/services/contract/ ./internal/adapters/secondary/postgres/ -run TestContractSold -count=1` | ❌ W0 | ⬜ pending |
| 11-03-01 | 03 | 1 | TICK-01 | T-11-04 / — | Ticket create (any employee), kind closed set, kind CHECK | unit + integration | `go test ./internal/core/services/ticket/ -run TestTicketCreate -count=1` | ❌ W0 | ⬜ pending |
| 11-03-02 | 03 | 1 | TICK-02 | T-11-05 / — | Full lifecycle incl. reopen; resolved blocked on non-terminal linked activities | unit | `go test ./internal/core/services/ticket/ -run TestTicketLifecycle -count=1` | ❌ W0 | ⬜ pending |
| 11-03-03 | 03 | 1 | TICK-03 | T-11-06 / — | Atomic triage: ticket→planned, activities created, all-or-nothing | integration | `go test ./internal/adapters/secondary/postgres/ -run TestTicketTriage -count=1` | ❌ W0 | ⬜ pending |
| 11-03-04 | 03 | 1 | TICK-04 | T-11-07 / — | Dismissal blocked with logged hours; note carries N | unit + integration | `go test ./internal/core/services/ticket/ -run TestDismissalGuard -count=1` | ❌ W0 | ⬜ pending |
| 11-03-05 | 03 | 1 | TICK-05 | T-11-08 / — | History append-only; no update/delete endpoints exist | integration + API contract | `go test ./internal/core/services/ticket/ -run TestTicketAudit -count=1` | ❌ W0 | ⬜ pending |
| 11-04-01 | 04 | 1 | FND-04 | — / — | Origin refs stored; read returns stored refs (fallback is Phase 13) | integration | covered by FND-01 read assertions | ❌ W0 | ⬜ pending |
| 11-05-01 | 05 | 1 | SC-9 | — / — | Migration up/down pairs + cycle tests 014–017 | migration cycle | `go test ./internal/adapters/secondary/postgres/ -run TestMigration014 -count=1` (…015/016/017) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/core/domain/ticket/` — ticket.go + errors.go (domain types, status/kind constants, transition matrix)
- [ ] `internal/core/domain/audit/` — general audit-log entity
- [ ] `internal/core/services/ticket/ticket_test.go` + `ticket_integration_test.go` — lifecycle/triage/guard/audit coverage
- [ ] `internal/adapters/secondary/postgres/ticket_repository_test.go` — triage tx, Σ hours, history read
- [ ] `internal/adapters/secondary/postgres/audit_log_repository_test.go`
- [ ] Migration cycle tests `TestMigration014..017` in the postgres package
- [ ] **Fix pre-existing red tests** `TestMigration011_ActivityOntology_UpDownUpCycle` + `TestMigration012_StaffingSchema_UpDownUpCycle` (seed wiring — Pitfall 3)
- [ ] `exported_test_helpers.go` teardown list extended with `tickets`, `ticket_comments`, `audit_logs`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ADR-P-003 revision + ADR-P-013 recorded in vault decisions folder | SC-9 (ADR part) | Doc artifact, not code | Confirm `hourglass-vault/decisions/project/ADR-P-003` reflects the revised boundary list and `hourglass-vault/decisions/project/ADR-P-013` exists with origin-axis decision record |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 300s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
