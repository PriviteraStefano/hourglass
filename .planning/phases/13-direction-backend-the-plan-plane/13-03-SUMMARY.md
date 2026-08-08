---
phase: 13-direction-backend-the-plan-plane
plan: 03
subsystem: domain-contracts
tags: [direction, org-settings, ports, testdata-mocks, domain, adr-be-018, contracts]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane (13-CONTEXT D-13-01..34, 13-RESEARCH, 13-PATTERNS, ADR-BE-018 vocab/encoding pins, 13-UI-SPEC API Data Contracts)
    provides: the compile-time contracts every later plan in the phase compiles against
provides:
  - direction domain: Direction entity (15 D-13-01 fields), status constants + pinned matrix (CanTransition/IsTerminalStatus), derived-state + claim-spectrum vocabularies, Warning/CoverageRow/PlanRow/DirectionRefs/AbsenceWindow shapes, audit vocabulary (AuditEntityDirection + 6 actions)
  - orgsettings domain: 4 key constants, mode + horizon vocabularies, DefaultDailyHours 8.0, knownKeys validators, audit pins, 4 sentinels
  - ports.DirectionRepository (9 methods) + ports.OrgSettingsRepository (Get/List/Upsert) — pinned signatures with in-tx audit contracts
  - testdata MockDirectionRepo + MockOrgSettingsRepo with compile-time port assertions
affects: [13-04 orgsettings service, 13-05 direction repo write side, 13-06 read side, 13-07 direction service, 13-08 origin fallback, Phase 19 surfaces, Phase 14 availability]

requirements-completed: [DIR-01, DIR-02, DIR-03, DIR-04, DIR-06]  # copied verbatim from PLAN.md; NOT marked complete in REQUIREMENTS.md — shared-ID gate: every ID is also declared by sibling plans 13-01/13-02/13-04..13-08 whose SUMMARYs don't all exist yet (requirements.ready-ids: 0/5 ready)

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens / 3 tasks / low confidence)
actuals:
  tokens: 8075    # chars/4 over the realized diff (32298 added chars across 8 files)
  tasks: 3        # tasks completed
  commits: 3      # commits made

# Tech tracking
tech-stack:
  added: []       # no package installs (Go module set unchanged)
  patterns:
    - "Ticket-domain house shape: entity + status consts + unexported transitionMatrix + CanTransition/IsTerminalStatus"
    - "Coverage-domain house shape: sentinel errors file + JSONNames map; exported audit vocabulary constants block (ADR-BE-018 pins)"
    - "Port doc-comments carry the in-tx audit contract + CR-01 lock semantics (ticket/coverage port analog)"
    - "Mock pattern: mutex + map + per-method Fn overrides + Audits capture + compile-time var _ ports.X assertion (mock_ticket_repo analog)"

key-files:
  created:
    - internal/core/domain/direction/direction.go
    - internal/core/domain/direction/errors.go
    - internal/core/domain/orgsettings/orgsettings.go
    - internal/core/domain/orgsettings/errors.go
    - internal/core/ports/direction_repository.go
    - internal/core/ports/org_settings_repository.go
    - internal/core/services/testdata/mock_direction_repo.go
    - internal/core/services/testdata/mock_org_settings_repo.go
  modified: []    # no pre-existing files touched

key-decisions:
  - "MockDirectionRepo absence-window stub field named Windows (setter SetAbsenceWindows): the plan's literal field name AbsenceWindows collides with the port method AbsenceWindows — Go forbids a field and a method with the same name on one type; the pinned setter surface is unchanged (13-07/13-08 seed via setters)"
  - "Port signatures follow the plan exactly where PATTERNS.md suggested alternatives: Coverage takes employeeIDs []uuid.UUID (scope resolution stays in the service, D-13-25), ListPlan takes employeeID *uuid.UUID, Create takes supersedesID + audits []*audit.AuditLog (two-row supersede tx)"
  - "No Unclaim port method: unclaim = Cancel of the claim row (D-13-16); the 9-method surface is the plan's pin"

# Coverage (#1602) — one entry per shipped deliverable
coverage:
  direction-domain:
    deliverable: "direction domain — entity, matrix, vocabularies, errors"
    verification:
      - kind: command
        ref: "go build ./internal/core/domain/direction/ && go vet ./internal/core/domain/direction/"
        status: pass
      - kind: command
        ref: "acceptance battery (temp test, deleted after run): CanTransition draft->active true, active->superseded false, IsTerminalStatus superseded/cancelled true, 10 sentinels + JSONNames"
        status: pass
    human_judgment: false
  orgsettings-and-ports:
    deliverable: "orgsettings domain (keys/validators/audit pins) + both repository ports"
    verification:
      - kind: command
        ref: "go build ./internal/core/domain/orgsettings/ ./internal/core/ports/ && go vet ./internal/core/domain/orgsettings/ ./internal/core/ports/"
        status: pass
      - kind: command
        ref: "acceptance battery (temp test, deleted after run): 4 known keys, DefaultDailyHours 8.0, validator rejections (0 hours, string hours, bad date, 'year' horizon, unknown mode), ErrUnknownKey on unknown keys"
        status: pass
    human_judgment: false
  testdata-mocks:
    deliverable: "MockDirectionRepo + MockOrgSettingsRepo (compile-time port contracts)"
    verification:
      - kind: command
        ref: "go build ./internal/core/services/testdata/ && go test ./internal/core/services/testdata/ -count=1"
        status: pass
      - kind: command
        ref: "behavioral battery (temp test, deleted after run): default Get org-scoped ErrDirectionNotFound, Create assigns ID, A8 claim row shape + Claims capture, Set* helpers drive read-methods, mock org-settings nil-when-absent + UpsertErr + audit capture"
        status: pass
    human_judgment: false
---

# Phase 13 Plan 3: Direction & Org Settings Domains, Ports and Testdata Mocks — Summary

Direction + orgsettings domains, both repository ports, and the testdata mocks: the compile-time contracts every 13-04..13-08 plan compiles against, with the ADR-BE-018 vocabularies (status, derived, claim-spectrum, audit, settings keys) enforced as exported constants.

**Duration:** ~10 min (2026-08-08T12:28Z → 12:37Z) · **Tasks:** 3/3 · **Commits:** 3 · **Files:** 8 created · **Estimate:** 30000 tokens → **Actual:** 8075 tokens

## Accomplishments

1. **Direction domain** (`internal/core/domain/direction/`) — `Direction` with the 15 D-13-01 fields and D-13-01 json tags; status constants with the pinned matrix (draft→active, draft→cancelled, active→cancelled; superseded reachable only via create-with-`supersedes_id` — D-13-08) exposed as `CanTransition`/`IsTerminalStatus`; derived-state vocabulary (`done`/`lapsed`/`claimed`), claim-spectrum constants (`not_claimed`/`partially_claimed`/`fully_claimed` — D-13-15, `fully_claimed` only when a budget is set); `Warning{Type,Message}` with the 4 closed types (`away`/`partial`/`over-capacity`/`invalid` — 13-UI-SPEC contract, message pre-rendered server-side); read-model shapes `CoverageRow{EmployeeID,Date,Capacity,Planned,Gap}` (D-13-26), `PlanRow` (Direction + Done/Lapsed/ClaimState — D-13-27), `DirectionRefs{AssignedBy,AssignedTo}` (D-13-32), `AbsenceWindow{EmployeeID,Kind,StartsOn,EndsOn,Hours}` (D-13-29); audit vocabulary `AuditEntityDirection` + `created`/`activated`/`cancelled`/`superseded`/`claimed`/`unclaimed` (ADR-BE-018 §3 pin). `errors.go`: 10 sentinels + JSONNames (coverage house shape; `ErrClaimOverBudget`→409, `ErrNotWgMember`→403, `ErrInvalidTarget` for the XOR fast-fail).
2. **Orgsettings domain + both ports** — `orgsettings.go`: `KeyPlanningDailyHours/Deadline/Horizon/Mode`, mode vocabulary (`manager_planned`/`self_planned`), horizon vocabulary (`day`/`week`/`month`), `DefaultDailyHours = 8.0`, the `knownKeys` validator map with per-key validation (number > 0 / `2006-01-02` parseable / closed horizon / closed mode) behind exported `IsKnownKey`/`ValidateKey` (D-13-18 code-enforced vocabulary), audit pins `AuditEntityOrgSettings` + `AuditActionSettingsUpdated`; `errors.go`: `ErrUnknownKey`/`ErrInvalidValue`/`ErrForbidden`/`ErrNotFound` + JSONNames. `ports/direction_repository.go`: the 9 pinned methods Get/Create (supersede-on-create, two audit rows)/Activate/Cancel/Claim (in-tx Σ guard under FOR UPDATE)/ListPlan/Coverage (employeeIDs slice — service resolves scope)/AbsenceWindows (declared+confirmed)/FirstDirectionRefs (nil-when-none), every mutator taking its audit row(s) for in-tx writes (BE-016). `ports/org_settings_repository.go`: Get (nil when absent)/List/Upsert (in-tx settings-updated audit).
3. **Testdata mocks** — `MockDirectionRepo` (mutex + `Directions` map org-scoped, `Claims` capture, per-method Fn overrides GetFn…FirstDirectionRefsFn, `Audits` capture, A8-shape default Claim, `SetPlanRows`/`SetCoverageRows`/`SetAbsenceWindows`/`SetDirectionRefs` helpers, `var _ ports.DirectionRepository` assertion) + `MockOrgSettingsRepo` (mutex + `Values` map, GetFn/UpsertErr knobs, `Audits` capture, nil-when-absent default Get, `var _ ports.OrgSettingsRepository` assertion).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Mock absence-window stub field renamed `Windows`**
- **Found during:** Task 3 build
- **Issue:** the plan's literal stub field name `AbsenceWindows` collides with the port method `AbsenceWindows` — Go forbids a field and a method with the same name on one type (compile error: "field and method with the same name").
- **Fix:** stub field named `Windows`; the pinned setter `SetAbsenceWindows` is unchanged (sibling tests seed via setters).
- **Files modified:** internal/core/services/testdata/mock_direction_repo.go
- **Verification:** `go build ./internal/core/services/testdata/` + behavioral battery pass
- **Commit:** d665318

### Out-of-Scope Discovery (NOT fixed — logged to deferred-items.md)

**2. [Scope boundary] Pre-existing 13-01 regression: TestMigration011_ActivityOntology_UpDownUpCycle fails on `make test`**
- **Found during:** plan-level `go test ./...`
- **Issue:** migration `021_direction_rows.up.sql` errors `relation "activities" does not exist` (42P01) inside the 011 cycle test — 13-01 added migrations 021/022 without extending the 011 test's skip list (its pre-state applies 000-010, then globs every remaining `migrations/*.up.sql` including 021, which FKs `activities`, a 011 table). Pre-existing since 13-01 (whose SUMMARY deferred full-suite to wave merge); the test file at the pre-13-03 HEAD has 0 skip entries for 021/022. Only this one test fails; `go build ./...`, `go vet ./...`, and all 13-03 packages + dependents pass.
- **Fix:** belongs to 13-01's scope (12-01 precedent `ae7f4a6`): add `021_direction_rows.up.sql` + `022_org_settings.up.sql` to the skip list in `internal/adapters/secondary/postgres/activity_ontology_migration_test.go`. Logged in `.planning/phases/13-direction-backend-the-plan-plane/deferred-items.md`; surfaced for the wave-merge gate.

**Total deviations:** 1 auto-fixed (Rule 1); 1 out-of-scope discovery logged. **Impact:** none on the 13-03 contracts; the full-suite gate at wave merge will trip on the 13-01 regression until the skip-list fix lands.

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./internal/core/domain/direction/ && go vet ./internal/core/domain/direction/` (T1 verify) | PASS |
| `go build ./internal/core/domain/orgsettings/ ./internal/core/ports/ && go vet ...` (T2 verify) | PASS |
| `go build ./internal/core/services/testdata/ && go test ./internal/core/services/testdata/ -run TestMockContracts -count=1` (T3 verify) | PASS (no test matches the name — package compiles + full package suite green; contract proven by compile-time assertions) |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | FAIL — pre-existing 13-01 regression, only `TestMigration011_ActivityOntology_UpDownUpCycle` (see Deviations) |
| Task acceptance batteries (temp tests, run then deleted) | PASS — all criteria verified per task |

## Success Criteria

- Direction + orgsettings domains encode the ADR-BE-018 vocabularies as exported constants — **met**
- Both ports pinned; all later plans compile against them without signature changes — **met** (signatures verified against 13-04..13-08 plan usage: `direction.Direction/Warning/CoverageRow/PlanRow/AbsenceWindow/DirectionRefs`, `orgsettings.Audit*`/`Mode*`/`DefaultDailyHours`, `ports.DirectionRepository/OrgSettingsRepository`, mock names)
- Testdata mocks contract-complete for the service unit tests (13-07) — **met** (per-method Fn overrides + setters + audit capture + compile-time assertions)

## Decisions Made

See `key-decisions` frontmatter — recorded in STATE.md via `state.add-decision` (MockDirectionRepo `Windows` field rename; port-signature choices pinned to the plan where PATTERNS.md suggested alternatives; no Unclaim method — Cancel serves unclaim per D-13-16).

## Issues Encountered

None blocking this plan. One out-of-scope pre-existing regression logged to `deferred-items.md` (13-01 skip-list gap, wave-merge gate will trip).

## Requirements

Plan declared DIR-01/DIR-02/DIR-03/DIR-04/DIR-06 — **none marked complete**: every ID is shared with sibling plans (13-04..13-08) that have no SUMMARY yet; the shared-ID gate (`requirements.ready-ids` → 0/5 ready) defers marking until the last declaring plan finishes.

## Next Phase Readiness

Ready for 13-04 (orgsettings service) — the orgsettings domain validators, audit pins and `ports.OrgSettingsRepository` it consumes are in place.

## Self-Check

- [x] 8 created files exist on disk
- [x] 3 commits exist (5b42857, f1f06b3, d665318) — all with plan-format messages
- [x] `go build ./...` + `go vet ./...` green; all 13-03 packages + compile dependents tested green
- [x] One pre-existing out-of-scope test failure documented (13-01 regression, deferred-items.md)
- [x] SUMMARY.md written; STATE.md/ROADMAP.md updated; final metadata commit follows

## Self-Check: PASSED
