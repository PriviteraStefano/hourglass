---
phase: 14
slug: availability-backend-absences-capacity
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-08
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + testify (require/assert) + testcontainers-go per-package PG suites |
| **Config file** | none — Go stdlib testing; suite setup via `internal/adapters/secondary/postgres/test_setup.go` (`SetupPackageContainer`) |
| **Quick run command** | `go test ./internal/core/domain/availability/ ./internal/core/services/availability/ -count=1` |
| **Full suite command** | `make test` (runs `go test -v ./...`) |
| **Estimated runtime** | ~60 seconds (Phase 13 close: 24 packages green in <60s) |

---

## Sampling Rate

- **After every task commit:** Run the quick run command on the package(s) touched by the task
- **After every plan wave:** Run `make test` (catches cross-package breaks — e.g. D-13-29 closure's effect on direction packages)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | AVAIL-01/02 | — | Migrations 023/024/025 up/down/up cycles; CHECK vocabularies + reject-reason 2VL asserted with 23514 + constraint name | integration | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration02[3-5]' -count=1` | ❌ W0 | ⬜ pending |
| 14-02-01 | 02 | 1 | AVAIL-02 | T-14g-07 | `models.Role.IsValid()` accepts `hr`; role gates compile; JWT claims carry it | unit | `go test ./internal/models/ -count=1` | ✅ existing file, ❌ new cases | ⬜ pending |
| 14-04-01 | 04 | 1 | AVAIL-02 | T-14g-10 | D-13-29 closure: direction warnings no longer fire for declared-only windows; confirmed windows still warn | integration | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration|Direction' -count=1` + `go test ./internal/core/services/direction/ ./internal/adapters/primary/http/ -run TestDirection -count=1` | ✅ existing, ❌ behavior change | ⬜ pending |
| 14-03-01 | 03 | 2 | AVAIL-01 | T-14g-01 | Declare valid window (holiday/permit/unavailable) → 200 `declared` | integration | `go test ./internal/adapters/primary/http/ -run TestAvailabilityHandler -count=1` | ❌ W0 | ⬜ pending |
| 14-03-02 | 03 | 2 | AVAIL-01 | T-14g-03 | Invalid input (bad kind, `ends_on < starts_on`, `hours > 99.99`, non-integer hours) → 400, never 500 | integration | same | ❌ W0 | ⬜ pending |
| 14-03-03 | 03 | 2 | AVAIL-01 | T-14g-02 | Overlapping window → 409 (declared+confirmed count; withdrawn/rejected do not, D-14-13) | integration | same | ❌ W0 | ⬜ pending |
| 14-03-04 | 03 | 2 | AVAIL-01 | T-14g-02 | Concurrent overlapping declares → exactly one succeeds (CR-01 race battery) | integration | `go test ./internal/adapters/secondary/postgres/ -run TestAvailabilityRepository -count=1` | ❌ W0 | ⬜ pending |
| 14-03-05 | 03 | 2 | AVAIL-01 | T-14g-01 | Medical declare requires `certificate_ref` + auto-confirms immediately (D-14-02/05) | integration | same | ❌ W0 | ⬜ pending |
| 14-03-06 | 03 | 2 | AVAIL-01 | — | Every window event writes `audit_logs` in-tx (`entity_type='availability_window'`); failed audit rolls back (BE-012) | integration | same | ❌ W0 | ⬜ pending |
| 14-05-01 | 05 | 3 | AVAIL-02 | T-14g-04 | Confirm authority (service): resolved unit manager → `confirmed`; non-manager 403; self-confirm when employee IS unit manager (D-14-04); only `declared` confirmable → 409 | unit | `go test ./internal/core/services/availability/ -count=1` | ❌ W0 | ⬜ pending |
| 14-05-02 | 05 | 3 | AVAIL-02 | T-14g-06 | Reject requires reason → 400 without; `rejected` + audit `{reason}`; rejected is terminal (D-14-08/09) | unit | same | ❌ W0 | ⬜ pending |
| 14-05-03 | 05 | 3 | AVAIL-02 | — | Withdraw declared-only (owner), terminal `withdrawn` + audit; non-owner 403 (D-14-10) | unit | same | ❌ W0 | ⬜ pending |
| 14-05-04 | 05 | 3 | AVAIL-02 | T-14g-13 | HR medical edit (PUT) + certificate attach (POST .../certificate) `hr`-gated; non-hr 403; edit on non-medical 400 (D-14-03/11) | unit | same | ❌ W0 | ⬜ pending |
| 14-06-01 | 06 | 3 | AVAIL-01/02 | — | Work schedules: contract_types templates + per-employee override; fallback chain resolution (D-14-27..29) | integration | `go test ./internal/adapters/primary/http/ -run 'TestContractType|TestAvailabilityMembership' -count=1` | ❌ W0 | ⬜ pending |
| 14-07-01 | 07 | 4 | AVAIL-01/02 | T-14g-05 | Windows read org-wide (any member) with `certificate_ref` + docs filtered out for non-hr/non-unit-manager (D-14-24) — service filtering matrix | unit | `go test ./internal/core/services/availability/ -run TestReadModels -count=1` | ❌ W0 | ⬜ pending |
| 14-07-02 | 07 | 4 | AVAIL-01/02 | — | Capacity: weekly hours per schedule, confirmed-only subtraction, declared advisory field, validity-excluded employees absent, workload Σ submitted+approved on activity subtree, per scope activity/wg/unit/org (D-14-20..23) | integration | `go test ./internal/adapters/secondary/postgres/ -run TestAvailabilityRepository -count=1` + `go test ./internal/core/services/availability/ -run TestReadModels -count=1` | ❌ W0 | ⬜ pending |
| 14-08-01 | 08 | 5 | AVAIL-02 | T-14g-04 | HTTP permission matrix: confirm/reject/withdraw/HR-edit/certificate gates over HTTP (non-manager 403, hr-not-manager 403, self-confirm 200, terminal 409, reason 400, cross-org 404, unauth 401) | integration | `go test ./internal/adapters/primary/http/ -run TestAvailabilityHandler -count=1` | ❌ W0 | ⬜ pending |
| 14-08-02 | 08 | 5 | AVAIL-01/02 | T-14g-05 | D-14-24 filtering e2e over HTTP (hr + unit manager see `certificate_ref`/docs, other members do not) + capacity endpoint per scope | integration | same | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/core/domain/availability/` — domain package (Window, vocab, transition matrix, sentinels, audit constants) + unit tests
- [ ] `internal/core/ports/availability_repository.go` — port (compile-time contract) + testdata mocks
- [ ] `internal/adapters/secondary/postgres/availability_repository_test.go` — migration cycle tests 023/024/025 + mutator/read-model batteries
- [ ] `internal/adapters/primary/http/availability_handler_test.go` — integration battery (permission matrix, sentinels, race test)
- [ ] `exported_test_helpers.go` — teardown list additions + `seedAvailabilityWindowWithCert`/`seedContractType` helpers (named WithCert — the Phase 13 `seedAvailabilityWindow` already exists in `direction_repository_test.go`)
- [ ] `internal/models/models.go` + `models_test.go` — `RoleHR` + validCases
- [ ] D-13-29 closure test updates in `direction_repository_test.go` / `direction_test.go` / `direction_handler_test.go` (declared-window seeds → confirmed; new no-warning-on-declared subtest)

---

## Manual-Only Verifications

All phase behaviors have automated verification (backend-only phase; API contract verified via integration tests, no UI).

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
