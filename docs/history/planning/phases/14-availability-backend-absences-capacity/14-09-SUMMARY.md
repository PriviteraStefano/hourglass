---
phase: 14-availability-backend-absences-capacity
plan: 09
subsystem: api
tags: [go, availability, medical, certificate-ref, data-integrity, float-epsilon]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity
    provides: UpdateMedical service + repo paths (14-05), WindowHoursValid exported helper (14-03)
provides:
  - CR-01 closure: date-only HR medical edits preserve certificate_ref from service boundary through the SQL UPDATE into the persisted row; empty-string refs rejected on the curation path (ErrCertificateRequired → 400) at both service and repo layers
  - WR-01 closure: WindowHoursValid accepts legal DECIMAL(4,2) cent hours (0.29/1.15/2.30) via epsilon tolerance while sub-cent values stay invalid
affects: [Phase 19 history reads, 14-08 handler batteries, Phase 20+ (Today surfaces)]

# Actuals (#2632) — pairs with the plan's `estimate` (16000 tokens) to calibrate future estimates.
actuals:
  tokens: 3020
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Absent-field-keeps-current-value at the service boundary (house style update-validates-present-fields) + belt-and-braces in-tx guard at the repo boundary (D-14-05 invariant structural at the DB boundary)"
    - "Float whole-cent validation uses epsilon tolerance (math.Abs(math.Round(h*100)-h*100) < 1e-9) instead of exact equality"

key-files:
  created: []
  modified:
    - internal/core/services/availability/availability.go
    - internal/core/services/availability/availability_test.go
    - internal/adapters/secondary/postgres/availability_repository.go
    - internal/adapters/secondary/postgres/availability_repository_test.go
    - internal/core/domain/availability/availability.go
    - internal/core/domain/availability/availability_test.go

key-decisions:
  - "Empty-string certificate_ref rejected at the service boundary (Declare parity, availability.go:106) AND nil/empty refused in-tx at the repo before UPDATE/audit — the D-14-05 invariant holds on every path (belt-and-braces per plan)"
  - "WindowHoursValid epsilon is < 1e-9 — the review's supplied shape — keeping 4.005/99.995 (|diff| = 0.5) invalid while accepting binary-inexact cent values"

patterns-established:
  - "Pattern 1: invariants that must survive every caller are enforced in-tx at the repo boundary, with the service fast-fail as UX parity"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "Date-only HR medical edits preserve certificate_ref from service boundary through the SQL UPDATE into the persisted row (CR-01 service + repo halves)"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_UpdateMedical_PreservesCertificateRef"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_UpdateMedical/date-only edit preserves certificate_ref through the UPDATE"
        status: pass
    human_judgment: false
  - id: D2
    description: "Empty-string certificate_ref rejected on the curation path with ErrCertificateRequired at both service and repo layers; refused repo edits leave row + audit untouched"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_UpdateMedical_PreservesCertificateRef/empty-string certificate_ref is rejected"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_UpdateMedical/nil certificate_ref on a medical window is refused in-tx"
        status: pass
    human_judgment: false
  - id: D3
    description: "WindowHoursValid accepts legal whole-cent partial-day hours (0.29/1.15/2.30) and still rejects sub-cent values; the declare API accepts legal partial-day hours"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/core/domain/availability/availability_test.go#TestWindowHoursValid"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-08-12
status: complete
---

# Phase 14: Plan 09 Summary

**CR-01/WR-01 gap closure: certificate_ref preserved on date-only HR medical edits (service copy + in-tx repo guard + empty-ref rejection), and WindowHoursValid accepts legal cent hours (0.29/1.15/2.30) via epsilon tolerance**

## Performance

- **Duration:** 15 min
- **Started:** 2026-08-12T09:02:00Z (approx)
- **Completed:** 2026-08-12T09:17:13Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Service `UpdateMedical` now copies the current `CertificateRef` into the update window (absent field keeps current value — D-14-05/D-14-11) and rejects empty-string refs with `ErrCertificateRequired` (Declare parity with availability.go:106) — CR-01's service half closed
- Repo `UpdateMedical` now refuses nil/empty `certificate_ref` IN-TX after the medical-only kind check, before the UPDATE and before the audit insert (a refused edit leaves no trace); the D-14-05 invariant is structural at the DB boundary — CR-01's repo half closed
- `WindowHoursValid` uses an epsilon tolerance (`math.Abs(math.Round(h*100)-h*100) < 1e-9`) instead of exact float equality — legal DECIMAL(4,2) cent values (0.29/1.15/2.30) now pass while the 99.99 ceiling and positivity bounds keep 4.005/99.995/0/100 invalid (WR-01)

## Task Commits

Each task was committed atomically (TDD RED → GREEN per task):

1. **Task 1: Service UpdateMedical — copy current CertificateRef + reject empty refs** -
   - `fb860b3` (test) add failing test for UpdateMedical cert_ref preservation
   - `10cacca` (feat) preserve certificate_ref on date-only medical edits
2. **Task 2: Repo UpdateMedical — in-tx certificate_ref guard** -
   - `bff8746` (test) add failing test for in-tx certificate_ref guard
   - `ee2371f` (feat) refuse nil/empty certificate_ref in-tx on medical edits
3. **Task 3: Domain WindowHoursValid — epsilon tolerance** -
   - `d6888b0` (test) add failing test for legal cent hours in WindowHoursValid
   - `efafecb` (feat) epsilon tolerance in WindowHoursValid for legal cent hours

## Files Created/Modified

- `internal/core/services/availability/availability.go` - UpdateMedical: copies `w.CertificateRef` into the update window; rejects empty-string refs with `ErrCertificateRequired` (Declare parity)
- `internal/core/services/availability/availability_test.go` - +`TestService_UpdateMedical_PreservesCertificateRef` (capture-based subtests: date-only preserves, empty-ref rejected)
- `internal/adapters/secondary/postgres/availability_repository.go` - UpdateMedical: in-tx nil/empty certificate_ref guard returning `ErrCertificateRequired` before UPDATE/audit; doc comment extended
- `internal/adapters/secondary/postgres/availability_repository_test.go` - +2 subtests on `TestAvailabilityRepository_UpdateMedical` (date-only round-trip via `SELECT certificate_ref`, nil-ref refusal with unchanged row + zero audit rows)
- `internal/core/domain/availability/availability.go` - `WindowHoursValid`: epsilon tolerance `< 1e-9` replacing exact float equality; doc comment updated
- `internal/core/domain/availability/availability_test.go` - `TestWindowHoursValid` valid list gains 0.29, 1.15, 2.30

## Decisions Made

- **Belt-and-braces placement of the D-14-05 invariant:** enforced at BOTH layers per the plan — service fast-fail (empty-string rejection, Declare parity) + in-tx repo guard (nil/empty refusal before UPDATE/audit). The repo guard makes the invariant structural at the DB boundary so no caller can ever NULL the medical ref.
- **Epsilon = 1e-9:** the review's supplied shape (`math.Abs(math.Round(h*100)-h*100) < 1e-9`). Sub-cent values have |diff| = 0.5, three orders of magnitude above the threshold, so the ceiling/positivity behavior is unchanged.
- **Test contract for subtest (a) in Task 1:** the seeded `WindowRows` is a struct copy, so the ref anchor is set on BOTH the WindowMap entry and `WindowRows[0]` — `getWindow` resolves through the Windows read.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all RED tests failed for the right reasons (captured window had nil ref; empty-ref path returned nil error; cent values failed float equality), GREEN passes in one iteration each.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 14-10 (next gap-closure plan) — the availability batteries (14-03 declare, 14-05 lifecycle, 14-07 read-model) all stay green
- The `{before, after}` audit on date-only edits now records `certificate_ref` unchanged in both halves — Phase 19 history reads see truthful records

---

*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-12*

## Self-Check: PASSED

- All 6 task commits (fb860b3, 10cacca, bff8746, ee2371f, d6888b0, efafecb) and the SUMMARY commit (2ba3e64) exist in git history
- All modified source files exist on disk
- Plan-level verification battery green: service package, postgres `TestAvailabilityRepository`, domain package, and `go build ./...`
