---
phase: 14-availability-backend-absences-capacity
plan: 03
subsystem: backend
tags: [go, postgres, availability, declare, overlap-guard, audit, CR-01]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity (plan 01)
    provides: migrations 012/023 availability_windows schema + seedAvailabilityWindowWithCert helper
  - phase: 14-availability-backend-absences-capacity (plan 02)
    provides: domain Window entity + status/kind/audit vocabularies, ports.AvailabilityRepository, MockAvailabilityRepo, models.RoleHR
provides:
  - The absence DECLARE vertical end-to-end (AVAIL-01): POST /availability/windows → handler → service (validation fast-fails + employee-self-or-hr gate, D-14-26) → repo (CR-01 overlap-guard tx under FOR UPDATE, D-14-15) → in-tx audit (BE-012, D-14-12)
  - availability_repository.go Declare: the authoritative overlap predicate (active-only declared+confirmed, inclusive range intersection), medical auto-confirm at insert (D-14-02), full-interface compile-time assertion with not-implemented stubs for the 14-05/06/07 surface
  - availability service package (7-arg constructor with the SHARED orgsettings + routing services, D-G parity) + WindowHoursValid export in the domain
  - availability handler with the sentinel map (404/400/403/409/500); route wired in cmd/server/main.go + the fixture
  - Repo batteries: concurrent race (exactly-one-winner), audit-rollback (BE-012), active-only exclusion (D-14-13), medical dual-audit
affects: [14-05 lifecycle mutators (replace the stubs), 14-06 contract types, 14-07 read models, 14-08 HTTP surface]

actuals:
  tokens: 15297   # chars/4 over realized diff (61190 chars, 10 files)
  tasks: 3
  commits: 9

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CR-01 overlap guard: SELECT ... FOR UPDATE on the user's ACTIVE overlapping windows inside the declare tx — exactly one concurrent winner (D-14-15)"
    - "In-tx audit write (BE-012): insertAvailabilityAudit helper; failed audit insert rolls back the window (T-14g-09)"
    - "Status derived from kind at the repo INSERT (medical → confirmed, D-14-02), never trusted from the caller"
    - "Not-implemented stubs + full-interface assertion: a plan that wires early compiles the whole port surface, and the stubs fail every later plan's RED tests cleanly"

key-files:
  created:
    - internal/adapters/secondary/postgres/availability_repository.go
    - internal/core/services/availability/availability.go
    - internal/adapters/primary/http/availability_handler.go
    - internal/adapters/primary/http/availability_handler_test.go
    - internal/adapters/secondary/postgres/availability_repository_test.go
    - internal/core/services/availability/availability_test.go
  modified:
    - cmd/server/main.go
    - internal/adapters/primary/http/handler_test_helper.go
    - internal/core/domain/availability/availability.go
    - internal/core/domain/availability/availability_test.go

key-decisions:
  - "Full-interface assertion lands in 14-03 via not-implemented stubs: the plan pins BOTH the assertion and the tracer fixture wiring in Task 1 while the port declares 14 methods owned by 14-05/06/07 — the Phase 13 deferral precedent (13-05) only worked because nothing wired the partial repo; the stubs fail every later plan's RED tests cleanly and no route can reach them"
  - "windowHoursValid exported as WindowHoursValid: the plan's service fast-fail names the domain helper, which 14-02 pinned unexported — exporting keeps single-source-of-truth (no re-implementation drift)"
  - "Service sets the observable status (declared/confirmed) before the repo call; the repo re-derives from kind authoritatively — the mock-based unit test caught the empty-status gap"
  - "The confirmed audit row for medical is repo-internal: the port's Declare takes ONE audit row, so the 'two audit rows with actor id' assertion lives at the repo boundary battery (Task 2), not the service mock (mechanically inexpressible there — same class as the 14-04 RED-placement deviation)"

patterns-established:
  - "Repo derives persisted status from kind; service mirrors it for mock-observable parity"
  - "Tolerant envelope decode in the handler-test helper: unregistered-route 404s fail the status assertion, not a JSON decode error — clean RED semantics"

requirements-completed: [AVAIL-01]

coverage:
  - id: D1
    description: "Declare end-to-end over HTTP through the fixture — holiday → 200 {status: declared} + in-tx declared audit row; validation 400s never 500 (bad kind, ends_on < starts_on, hours 100.00, sub-cent 4.005, empty body, missing dates); overlap 409 + adjacency edge 200; medical without certificate_ref 400 (D-14-05), with ref 200 {status: confirmed} + declared + confirmed audit rows"
    requirement: AVAIL-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler"
        status: pass
    human_judgment: false
  - id: D2
    description: "Repo declare correctness — concurrent overlapping declares: exactly one winner + ErrOverlap loser with one window + one audit row surviving (CR-01/D-14-15); audit-rollback: FK-forced audit failure leaves no window row (BE-012/T-14g-09); active-only overlap: withdrawn/rejected never block, declared/confirmed always do (D-14-13); medical: status confirmed + certificate_ref persisted + both audit rows with the actor id (D-14-02/05/12)"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_DeclareConcurrentRace"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_DeclareAuditRollback"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_DeclareActiveOnly"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_DeclareMedicalAutoConfirm"
        status: pass
    human_judgment: false
  - id: D3
    description: "Service declare unit tests — role gates (employee-self ok + audit capture, hr-for-anyone ok, employee-for-other ErrForbidden with zero repo calls, D-14-26); table-driven validation fast-fails each with its sentinel and never reaches the repo; medical branch sets confirmed status + declared audit DTO with ActorID/EntityID pinned; repo ErrOverlap propagates (CR-01 parity)"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_Declare_RoleGates"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_Declare_ValidationFastFails"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_Declare_MedicalAutoConfirm"
        status: pass
    human_judgment: false
  - id: D4
    description: "Wiring — POST /availability/windows auth-gated in cmd/server/main.go + the fixture mirror (shared orgsettings + routing services, D-G parity); repo-wide build green"
    verification:
      - kind: other
        ref: "go build ./... (exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 17min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 03: Absence Declare Tracer — Overlap-Guard Transaction + Medical Auto-Confirm Summary

**The absence DECLARE vertical is proven end-to-end: POST /availability/windows through the CR-01 overlap-guard transaction (the user's ACTIVE windows locked FOR UPDATE in-tx — the race battery proves exactly one concurrent winner), the service's validation fast-fails + employee-self-or-hr gate (D-14-26), the medical auto-confirm branch (D-14-02) with certificate_ref required (D-14-05), and the in-tx audit rows (declared always, confirmed for medical, D-14-12) — the phase's transaction shapes are proven and 14-05 expands them mechanically.**

## Performance

- **Duration:** 17 min
- **Started:** 2026-08-11T13:23:43Z
- **Completed:** 2026-08-11T13:40:18Z
- **Tasks:** 3 (each TDD: RED + GREEN)
- **Files modified:** 10 (6 created, 4 modified)

## Accomplishments

- **Repo Declare (the correctness heart):** `BeginTx` → authoritative overlap guard `SELECT aw.id FROM availability_windows aw WHERE org_id AND user_id AND status IN ('declared','confirmed') AND starts_on <= $4::date AND ends_on >= $3::date ORDER BY id LIMIT 1 FOR UPDATE` → any row `ErrOverlap` (409). The predicate mirrors the shipped AbsenceWindows shape inverted boundary-correct: inclusive intersection (a window ending the day before another starts does NOT overlap, D-14-14) and active-only (withdrawn/rejected never block, D-14-13). The INSERT derives status from the kind — medical → `confirmed` (D-14-02), everything else `declared`. `insertAvailabilityAudit` writes in the SAME tx (BE-012, D-14-12): the declared row always, the confirmed row when medical — a failed audit insert rolls back the window (T-14g-09). In-tx re-read + commit (direction Create skeleton, 13-05).
- **Race-closed (CR-01, D-14-15):** the concurrency battery races two overlapping declares for one user through a start channel + buffered results — the outcome set is deterministically `{success, ErrOverlap}` with exactly one window row and one audit row surviving. The FOR UPDATE serialization is the lock that makes the plan's must-have "exactly one succeeds" true.
- **Audit-atomic (BE-012):** the rollback battery forces the in-tx audit insert to fail (actor id with no users row → audit_logs actor_id FK, 23503) and asserts the window row is NOT persisted — a failed audit insert cannot suppress the audit and keep the state.
- **Service:** 7-arg constructor with the SHARED orgsettings + routing services (D-G parity, main.go wiring order). `Declare` fast-fails validation first (closed kind set, zero/absent dates, ends-on-before-starts, DECIMAL(4,2) hours ceiling + whole-cent via `WindowHoursValid`, medical certificate_ref required — D-14-05), then the D-14-26 gate (actor == window user OR role `hr`; employee-for-other → 403 with zero repo calls), then the declared audit DTO (ActorID = actor, EntityID = the generated window id) into the repo.
- **Handler + wiring:** `POST /availability/windows` auth-gated with the sentinel map (ErrWindowNotFound 404; invalid/hours/kind/certificate/reason/not-medical 400; forbidden 403; overlap/invalid-transition 409; default 500 — client input never 500s). Wired in cmd/server/main.go AND the fixture mirror (same shape as the direction block), so the battery exercises the real stack.
- **The integration battery is the phase tracer:** declare-holiday → 200 `status: declared` + in-tx audit row observable in DB state; six validation 400s; overlap 409 + adjacency 200; medical 400-without-ref → 200 confirmed with BOTH audit rows. All proven over HTTP through the fixture.

## Task Commits

Each task was committed atomically (TDD: test commit then feat commit):

1. **Task 1 (TRACER): Declare vertical** — `8ac58f0` (test: failing battery), `e018636` (feat: repo Declare), `8b46fd1` (feat: service + domain export), `eeb2ffd` (feat: handler), `1dc8318` (feat: wiring + port surface)
2. **Task 2: Repo batteries** — `def1cb4` (test: race/rollback/active-only/medical; green against the shipped repo — plan-sanctioned)
3. **Task 3: Service unit tests** — `bf896d5` (test: unit battery — RED on the empty-status gap), `7d5dfda` (feat: declared status in the service)
4. **Formatting:** `24c33ce` (style: gofmt realign)

**Plan metadata:** committed after this file

## Files Created/Modified

- `internal/adapters/secondary/postgres/availability_repository.go` - Declare tx (overlap guard FOR UPDATE, INSERT kind-derived status, in-tx audit, re-read, commit) + scanAvailabilityWindowRow/getWindowByIDTx/insertAvailabilityAudit + 14 not-implemented stubs + full-interface assertion (created)
- `internal/core/services/availability/availability.go` - availabilitysvc package: 7-arg constructor, DeclareRequest DTO, Declare (validation fast-fails, D-14-26 gate, medical branch, audit DTO) (created)
- `internal/adapters/primary/http/availability_handler.go` - AvailabilityHandler + Declare + writeError sentinel map (created)
- `internal/adapters/primary/http/availability_handler_test.go` - TestAvailabilityHandler declare battery (created)
- `internal/adapters/secondary/postgres/availability_repository_test.go` - four repo batteries (created)
- `internal/core/services/availability/availability_test.go` - service unit tests with the mock fixture (created)
- `cmd/server/main.go` - availability wiring + POST /availability/windows route (modified)
- `internal/adapters/primary/http/handler_test_helper.go` - fixture availability block mirroring main.go (modified)
- `internal/core/domain/availability/availability.go` - windowHoursValid → WindowHoursValid export (modified)
- `internal/core/domain/availability/availability_test.go` - helper references updated (modified)

## Decisions Made

- **Full-interface assertion via not-implemented stubs (not deferral):** the plan pins BOTH the compile-time assertion AND the tracer fixture wiring in Task 1, while the port declares 14 methods owned by 14-05/06/07. The Phase 13 deferral precedent (13-05) only worked because nothing wired the partial repo. The stubs return explicit not-implemented errors: no route can reach them, and they fail every later plan's RED tests cleanly (a `ErrWindowNotFound`-style stub would have falsely greened some 14-05 RED assertions).
- **`WindowHoursValid` exported:** the plan's service fast-fail names the domain helper, which 14-02 pinned unexported (a compile blocker the service would otherwise re-implement — drift risk the phase explicitly avoids).
- **Service mirrors the observable status; repo stays authoritative:** the service sets `declared`/`confirmed` on the window before the repo call (mock-observable parity; the RED unit test caught the empty-status gap), while the repo INSERT re-derives the persisted status from the kind — a caller can never write a status it doesn't own.
- **Medical "two audit rows" proven at the repo boundary:** the port's Declare takes ONE audit row, so the service-level two-row assertion is mechanically inexpressible (mock captures one row); the confirmed row is repo-internal and asserted in `TestAvailabilityRepository_DeclareMedicalAutoConfirm` (declared + confirmed, both with the actor id) — same class as the 14-04 RED-placement deviation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Full port surface must compile for the pinned assertion + wiring**
- **Found during:** Task 1 (wiring step — `availabilitysvc.NewService` demanded `ports.AvailabilityRepository`, the partial repo did not satisfy it)
- **Issue:** The plan pins the full-interface assertion AND the fixture/main.go wiring in the same plan, but the repo only implements Declare; the Phase 13 deferral precedent could not apply because this plan wires the real repo into the tracer battery.
- **Fix:** Added 14 not-implemented stub methods (explicit `notImplemented(method)` errors) + the `var _ ports.AvailabilityRepository` assertion. Unreachable (no routes), and they fail every 14-05/06/07 RED test cleanly.
- **Files modified:** internal/adapters/secondary/postgres/availability_repository.go
- **Verification:** `go build ./...` exit 0; full postgres + http packages green.
- **Committed in:** 1dc8318

**2. [Rule 3 - Blocking] windowHoursValid pinned unexported, service needs it**
- **Found during:** Task 1 (service compile)
- **Issue:** The plan's service validation step names the domain helper `windowHoursValid`, but 14-02 pinned it unexported — the service could not call it without re-implementing the DECIMAL(4,2) ceiling logic (drift risk).
- **Fix:** Exported it as `WindowHoursValid` (additive visibility change, same body); updated the domain test references.
- **Files modified:** internal/core/domain/availability/availability.go, availability_test.go
- **Verification:** domain package tests green.
- **Committed in:** 8b46fd1

**3. [Rule 1 - Bug] Service returned an empty status for non-medical declares**
- **Found during:** Task 3 (RED run — `expected "declared", actual ""`)
- **Issue:** The service only set `Status` in the medical branch; holiday declares left the window with an empty status at the service boundary (the mock captures the window verbatim; only the repo re-read derived 'declared').
- **Fix:** The service now sets `Status = StatusDeclared` by default and `StatusConfirmed` for medical — mirroring the repo's authoritative kind-derived status.
- **Files modified:** internal/core/services/availability/availability.go
- **Verification:** service unit tests green; handler battery still green.
- **Committed in:** 7d5dfda

**4. [Rule 3 - Blocking] Test seed: rejected window violated the 023 2VL CHECK**
- **Found during:** Task 2 (RED run — 23514 `availability_windows_reject_reason_check`)
- **Issue:** `seedAvailabilityWindowWithCert` (14-01) has no rejection_reason param, so the seeded 'rejected' row tripped the never-NULL-satisfiable guard.
- **Fix:** Seeded the rejected row inline with `rejection_reason` set; the shared helper stays untouched.
- **Files modified:** internal/adapters/secondary/postgres/availability_repository_test.go
- **Verification:** active-only battery green.
- **Committed in:** def1cb4

---

**Total deviations:** 4 auto-fixed (3 Rule 3 blocking, 1 Rule 1 bug)
**Impact on plan:** All fixes were required to land the tracer's compile surface and to keep the batteries honest. No scope creep — the stubs are explicitly owned by later plans and fail their REDs cleanly.

## Issues Encountered

- **Plan Task 3's "two audit rows" service assertion is repo-layer:** the port's Declare signature (pinned 14-02) takes one audit row and the mock captures one; the service unit test asserts the declared DTO (action/ActorID/EntityID/EntityType) + confirmed status, and the two-row proof lives at the repo boundary (Task 2 battery 4). Documented in Decisions, not a defect.
- **Gofmt drift between commits:** the gofmt realign of the domain Window struct landed after its commit; caught by `gofmt -l` before close-out and committed as a style commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-05 (lifecycle mutators):** Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate replace the not-implemented stubs — the Activate/cancelWithGuard skeletons (CR-01) and insertAvailabilityAudit are proven here; the shared routing service is already in the constructor for the unit-manager confirm authority (D-14-01).
- **14-06 (contract types):** the CRUD stubs slot into the same file; the orgsettings service is wired for the default-schedule resolution (D-14-18).
- **14-07 (read models):** Windows/Capacity/Attachments stubs + `idx_availability_windows_org_user_dates` ready; the declared-advisory field rides on the shipped predicate shapes.
- **14-08 (HTTP surface):** the remaining `/availability` routes mirror the direction route block; the fixture pattern is established.
- The full suite is green (`go vet ./...` clean; postgres 38s, http 55s, services green) — the declare tracer changed nothing outside its packages.

---

*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*
