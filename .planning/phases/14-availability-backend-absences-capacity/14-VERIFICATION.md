---
phase: 14-availability-backend-absences-capacity
verified: 2026-08-12T11:45:00Z
status: passed
score: 12/12 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 9/12
  gaps_closed:
    - "HR medical curation preserves certificate_ref: absent fields keep the current values (D-14-05/D-14-11) — a date-only medical edit must NOT null certificate_ref (CR-01)"
    - "Employee can declare a legal whole-cent partial-day window (0.29h, 2.30h) via the API (AVAIL-01) — valid DECIMAL(4,2) values must not 400 (WR-01)"
    - "Capacity workload is period-scoped: workload = Σ submitted+approved entries on the activity subtree for the requested period (D-14-19) — never a lifetime sum (WR-02)"
    - "Never 500 for client input: the full /availability surface maps client errors to 4xx (T-14g-03/18/25) (WR-03)"
  gaps_remaining: []
  regressions: []
---

# Phase 14: Availability Backend — Absences + Capacity Verification Report

**Phase Goal:** Absence lifecycle works server-side over the shipped availability_windows schema (declare → confirm/reject, HR medical curation), plus derived capacity queries (weekly hours − confirmed absences) with workload from submitted+approved entries (AVAIL-01/02, supports AVAIL-04 and Phase 13's DIR-05).
**Verified:** 2026-08-12T11:45:00Z
**Status:** passed
**Re-verification:** Yes — gap-closure round (plans 14-09/14-10/14-11 closed CR-01, WR-01, WR-02, WR-03)

## Summary

The previous verification (14-VERIFICATION.md, 2026-08-11, `gaps_found`, 9/12) found 4 must-have truth failures, all confirmed unfixed at that time. This round re-verifies against the current tree after gap-closure plans 14-09 (CR-01 + WR-01), 14-10 (WR-02), and 14-11 (WR-03) executed. **All 4 gaps are closed in code, with regression tests at the layer that owns each invariant, and every previously-passed truth remains green.** Score is now 12/12.

## Goal Achievement

### Observable Truths

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | Migrations 023/024/025 exist append-only from 023, with up/down pairs, named CHECK constraints | ✓ VERIFIED (regression) | `TestMigration02[3-5]` PASS in this run (8.9s combined battery) |
| 2 | Domain pins the closed vocabulary/matrix/audit constants matching the 023 CHECK | ✓ VERIFIED (regression) | `go test ./internal/core/domain/availability/` PASS (0.40s) |
| 3 | ports.AvailabilityRepository + MockAvailabilityRepo compile; models.RoleHR | ✓ VERIFIED (regression) | `go build ./...` clean; untouched by gap round |
| 4 | Declare via POST /availability/windows: 200/400/409 semantics, medical auto-confirm | ✓ VERIFIED (regression) | `TestAvailabilityHandler` PASS (13.3s) incl. `hours over 99.99 ceiling → 400` subtest |
| 5 | Confirm/reject lifecycle: authority matrix, reason required, terminal 409s, in-tx audit | ✓ VERIFIED (regression) | HTTP + repo lifecycle batteries PASS |
| 6 | **CR-01:** HR medical curation preserves certificate_ref — date-only edit must NOT null it | ✓ VERIFIED (closed) | Service copies `w.CertificateRef` (availability.go:334) + empty-ref rejection (344-347); repo in-tx guard refuses nil/empty before UPDATE/audit (availability_repository.go:424-430). Behavioral: `TestService_UpdateMedical_PreservesCertificateRef` PASS (capture-based); repo subtests "date-only edit preserves certificate_ref through the UPDATE" (row re-read via `SELECT certificate_ref`) + "nil certificate_ref refused in-tx" (row unchanged, zero audit rows) PASS (3.1s) |
| 7 | **WR-02:** Capacity workload period-scoped — Σ submitted+approved on subtree for the requested period | ✓ VERIFIED (closed) | Workload CTE now carries `AND te.entry_date >= $3::date AND te.entry_date < $4::date + INTERVAL '1 day'` (availability_read_models.go:146) — identical to sibling columns. Behavioral: `TestAvailabilityRepository_Capacity_WorkloadPeriodBound` PASS — in-period 6h+4h = 10.0 on every day row, out-of-period 5h+7h excluded (would be 22.0 unfixed), userB = 0.0; HTTP regression seeds 2026-10-03 entry outside 11-02..11-08 period, workload_hours stays 4.0 |
| 8 | Windows read org-wide with filters/ordering/pagination + D-14-24 server-side medical privacy | ✓ VERIFIED (regression) | HTTP privacy e2e PASS in `TestAvailabilityHandler` |
| 9 | ResolveSchedule fallback chain + hr-gated contract-type CRUD + membership endpoint | ✓ VERIFIED (regression) | `TestContractTypesHandler` PASS (5.8s) |
| 10 | **WR-03:** Never 500 for client input on /availability | ✓ VERIFIED (closed) | Service ceiling `HoursPerPeriod > 999.99 → ErrInvalidRequest` (contract_types.go:57-59); `ports.ErrInvalidRequest` sentinel (errors.go:9); `wrapPGError` case "22003" (postgres.go:32-33); writeError 400 case (availability_handler.go:92). Behavioral: HTTP subtest `hours above the DECIMAL(5,2) ceiling returns 400 (WR-03)` PASS — asserts 400, never 500; `TestWrapPGError_22003` PASS (22003 mapped, 23514 passthrough); boundary 999.99 accepted |
| 11 | **WR-01:** Legal whole-cent partial-day windows (0.29h, 2.30h) declare-able via the API | ✓ VERIFIED (closed) | `WindowHoursValid` epsilon form `math.Abs(math.Round(h*100)-h*100) < 1e-9` (domain availability.go:210); declare fast-fail consumes the same exported helper (service availability.go:103). Behavioral: `TestWindowHoursValid` PASS — 0.29/1.15/2.30 valid, 4.005/99.995/0/100 invalid |
| 12 | Direction scheduler read path confirmed-only closure (D-14-21/DIR-05); full suite green | ✓ VERIFIED (regression) | `TestDirectionRepository` PASS in this run; `go build ./...` clean; full service package PASS (0.22s) |

**Score:** 12/12 truths verified (4 closed from previous round; 8 re-verified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/core/services/availability/availability.go` | UpdateMedical copies CertificateRef + rejects empty refs | ✓ VERIFIED | Lines 331-348 — absent field keeps current value (D-14-05/D-14-11) |
| `internal/adapters/secondary/postgres/availability_repository.go` | In-tx certificate_ref guard on UpdateMedical | ✓ VERIFIED | Lines 424-430 — nil/empty refused before UPDATE and audit insert |
| `internal/core/domain/availability/availability.go` | WindowHoursValid epsilon tolerance | ✓ VERIFIED | Line 210 — `math.Abs(math.Round(h*100)-h*100) < 1e-9` |
| `internal/adapters/secondary/postgres/availability_read_models.go` | Workload CTE period predicate | ✓ VERIFIED | Line 146 — `$3`/`$4` bounds identical to declared/partial_abs/full_abs |
| `internal/core/services/availability/contract_types.go` | validateContractType 999.99 ceiling | ✓ VERIFIED | Lines 53-59 — mirrors migration 024 DECIMAL(5,2) max |
| `internal/core/ports/errors.go` | ErrInvalidRequest sentinel | ✓ VERIFIED | Line 9 — additive |
| `internal/adapters/secondary/postgres/postgres.go` | wrapPGError 22003 → ports.ErrInvalidRequest | ✓ VERIFIED | Lines 32-33 |
| `internal/adapters/primary/http/availability_handler.go` | writeError maps ports.ErrInvalidRequest → 400 | ✓ VERIFIED | Line 92 in the 400 branch |
| Regression tests (6 files) | RED→GREEN pairs for each fix | ✓ VERIFIED | All present and PASS (detailed per truth above) |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Service UpdateMedical | Repo UPDATE | ref-bearing update window | ✓ WIRED | Service always passes current ref (availability.go:334); repo guard (424-430) makes NULLing impossible for any caller |
| Empty-string rejection | Declare's fast-fail | same `ErrCertificateRequired` sentinel | ✓ WIRED | availability.go:106 (Declare) ↔ 344-347 (UpdateMedical) — one invariant on both paths |
| WindowHoursValid | Declare service fast-fail | exported shared helper | ✓ WIRED | service availability.go:103 consumes the domain helper |
| Workload CTE | $3/$4 period args | same binding as sibling CTEs | ✓ WIRED | availability_read_models.go:146 verbatim sibling shape |
| validateContractType | Migration 024 DECIMAL(5,2) | 999.99 ceiling mirrors column max | ✓ WIRED | contract_types.go:57-59 |
| wrapPGError 22003 | availability writeError | ports.ErrInvalidRequest sentinel | ✓ WIRED | postgres.go:32-33 → handler 400 case (availability_handler.go:92) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Capacity workload | workload_hours | time_entries CTE (submitted+approved), **period-bounded** | ✓ Real data, correct scope | ✓ FLOWING (was HOLLOW — lifetime sum) |
| Capacity absence | confirmed_absence_hours | confirmed-only absences | ✓ Period-bounded via $3/$4 | ✓ FLOWING |
| Medical row | certificate_ref | date-only edit round-trip | ✓ Ref survives UPDATE (row re-read asserted) | ✓ FLOWING (was NULL-write) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Repo-wide compile | `go build ./...` | clean | ✓ PASS |
| CR-01 service half (capture-based preserve + empty-ref rejection) | `go test ./internal/core/services/availability/ -run 'TestService_UpdateMedical|TestContractTypes' -count=1` | ok | ✓ PASS |
| CR-01 repo half + WR-02 repo battery | `go test ./internal/adapters/secondary/postgres/ -run 'TestAvailabilityRepository_UpdateMedical|TestAvailabilityRepository_Capacity' -count=1` | ok (3.8s) | ✓ PASS |
| WR-01 domain epsilon cases | `go test ./internal/core/domain/availability/ -count=1` | ok (0.40s) | ✓ PASS |
| WR-03 adapter mapping | `go test ./internal/adapters/secondary/postgres/ -run TestWrapPGError -count=1` | ok (0.6s) | ✓ PASS |
| WR-02 + WR-03 HTTP boundary | `go test ./internal/adapters/primary/http/ -run 'TestContractTypesHandler|TestAvailabilityHandler' -count=1` | ok (19.6s) | ✓ PASS |
| Full service package | `go test ./internal/core/services/availability/ -count=1` | ok (0.22s) | ✓ PASS |
| Regression: migrations + direction confirmed-only | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration02[3-5]|TestDirectionRepository' -count=1` | ok (9.0s) | ✓ PASS |

### Probe Execution

No probe scripts declared for this phase (backend Go phase; verification via `go test`/`go build`). SKIPPED.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| AVAIL-01 | 14-01..14-04, 14-06..14-11 | Employee declares absence with type/date range; invalid/overlapping rejected | ✓ SATISFIED | Declare vertical + batteries green; WR-01 (legal cent values) and WR-03 (500 path) edge defects closed this round |
| AVAIL-02 | 14-01, 14-02, 14-04..14-11 | Manager/HR confirm/reject; rejects carry reason; HR curates medical with certificate_ref | ✓ SATISFIED | Lifecycle batteries green; CR-01 (date-only edit wiping certificate_ref) closed this round |

Orphaned requirements: none — both phase requirement IDs (AVAIL-01, AVAIL-02) appear in plan frontmatter across all 11 plans; AVAIL-03/04/05 map to Phase 16 per REQUIREMENTS.md (not this phase).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers (TBD/FIXME/XXX) in gap-closure modified files | — | None — scan clean across all 8 modified source files |
| internal/adapters/secondary/postgres/availability_repository.go | 726-769 | DeleteContractType ignores org-default key (WR-04) | ℹ️ Info | Pre-existing review warning, NOT in gap-closure scope (no must-have truth failed); still open |
| internal/core/services/availability/read_models.go | 352-392 | unit/wg scope not org-guarded (WR-05) | ℹ️ Info | Pre-existing review warning, NOT in gap-closure scope; still open |
| internal/adapters/primary/http/availability_handler.go | 255-264, 310-312 | Client-controlled content-type served verbatim (WR-06) | ℹ️ Info | Pre-existing review warning, NOT in gap-closure scope; still open |

Note: WR-04/WR-05/WR-06 were documented in 14-REVIEW.md and the previous VERIFICATION.md as warnings not tied to must-have truths; the gap-closure round scheduled only the four failing truths (CR-01, WR-01, WR-02, WR-03). They remain open but are non-blocking for this phase's goal and do not affect the status.

### Human Verification Required

None — every behavior-dependent truth has a passing behavioral test run in this session: CR-01 (row-level `SELECT certificate_ref` round-trip + in-tx refusal), WR-01 (epsilon acceptance cases), WR-02 (in/out-of-period seeded entries through real SQL + HTTP boundary), WR-03 (HTTP 400 assertion + SQLSTATE mapping unit test).

### Gaps Summary

**All 4 previously-failing must-have truths are closed with committed code and passing regression tests:**

1. **CR-01 (data loss):** service `UpdateMedical` copies `w.CertificateRef` into the update window and rejects empty-string refs; repo refuses nil/empty refs in-tx before the UPDATE and audit insert. Proven by capture-based service test + repo subtests that re-read the persisted row.
2. **WR-01:** `WindowHoursValid` uses epsilon tolerance; 0.29/1.15/2.30 accepted, sub-cent values still rejected. The declare path consumes the same helper.
3. **WR-02:** workload CTE carries the `$3`/`$4` period predicate identical to its sibling columns; repo battery proves out-of-period entries contribute 0 (10.0, not 22.0); HTTP boundary regression proves the same end-to-end.
4. **WR-03:** `validateContractType` 999.99 ceiling + `ports.ErrInvalidRequest` + `wrapPGError` 22003 mapping + writeError 400 case. HTTP asserts 400 (never 500) for `hours_per_period: 1000`; boundary 999.99 accepted.

All TDD commits present in git history (14-09: fb860b3→efafecb; 14-10: cdf6931→e736c0a/091c9fa; 14-11: 24fbb91→c1e5f3e; 22 commits after the previous verification's HEAD 87cd7fc). No regressions observed in the 8 previously-verified truths. `go build ./...` clean; no debt markers.

**Phase goal achieved.** Requirements AVAIL-01 and AVAIL-02 satisfied.

---

_Verified: 2026-08-12T11:45:00Z_
_Verifier: the agent (gsd-verifier)_
