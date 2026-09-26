---
phase: 14-availability-backend-absences-capacity
plan: 05
subsystem: backend
tags: [go, postgres, availability, lifecycle, confirm, reject, withdraw, certificate, CR-01, audit]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity (plan 03)
    provides: the Declare tx + insertAvailabilityAudit + not-implemented stubs the lifecycle mutators replace; the shared routing/orgsettings service fixture
  - phase: 14-availability-backend-absences-capacity (plan 02)
    provides: domain Window/Attachment entities, the status transition matrix + terminal vocabulary, ports.AvailabilityRepository signatures, MockAvailabilityRepo
  - phase: 14-availability-backend-absences-capacity (plan 01)
    provides: migration 023 status vocabulary + rejection_reason 2VL CHECK, migration 025 certificate_attachments BYTEA table, seedAvailabilityWindowWithCert helper
  - phase: 13-direction-backend-the-plan-plane (plan 13-06)
    provides: the routing.ResolveUnitManager confirm-authority resolution (shared instance, D-G parity)
provides:
  - Five repo mutators (Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate) with the CR-01 in-tx matrix re-check under FOR UPDATE, status-precondition UPDATE backstop, and in-tx audit (BE-012 — failed audit rolls back the state change)
  - Service orchestration: unit-manager confirm/reject authority via the shared routing service, self-confirm (D-14-04), HR-never-confirms (D-14-03), owner-only withdraw (D-14-10), HR-only edit/attach gates (D-14-11/06), reject reason fast-fail (D-14-09)
  - Repo batteries: transition matrix + terminal guards, reject reason at the repo boundary, BYTEA certificate round-trip, append-only attach, lifecycle + curation audit rollback proofs
  - Service unit batteries: the full authority matrix (manager/self/hr/non-manager/no-manager), HR gates, audit DTO capture via the mock
  - MockUnitRepo.ListMembershipsForUser now derives memberships from UnitMembers (the port contract) — consumed by the availability authority resolution
affects: [14-06 contract-type CRUD (stubs remain), 14-07 read-models (Windows production read the service authority resolution depends on), 14-08 HTTP surface (handler battery over the lifecycle), Phase 19 history filters (audit vocabulary)]

actuals:
  tokens: 16513    # chars/4 over realized diff (66055 chars, 5 files, 6 commits)
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CR-01 lifecycle transition: BeginTx → SELECT status FOR UPDATE (org-scoped, ErrNoRows → ErrWindowNotFound) → in-tx matrix re-check → UPDATE with locked status as WHERE precondition backstop → audit in-tx → re-read → commit (the Activate/cancelWithGuard skeleton)"
    - "In-tx audit atomicity proven by rollback battery: a failed audit insert (bogus actor → 23503 FK) rolls back the state change (BE-012, T-14g-15)"
    - "Service authority resolution via the SHARED routing service: ListMembershipsForUser → ResolveUnitManager per unit, any-of semantics, self-confirm falls out of the same check (D-G parity, no re-implementation)"
    - "Mock derives port semantics instead of hard-coding: ListMembershipsForUser scans UnitMembers — tests seed one map, both ListMembers and the membership lookup serve the routing walk"

key-files:
  created: []
  modified:
    - internal/adapters/secondary/postgres/availability_repository.go
    - internal/adapters/secondary/postgres/availability_repository_test.go
    - internal/core/services/availability/availability.go
    - internal/core/services/availability/availability_test.go
    - internal/core/services/testdata/mocks.go

key-decisions:
  - "Service resolves the window owner + kind via the org-wide Windows read (the port surface has no single-window getter; the pinned port cannot change) — the production read lands in 14-07, the mock serves it now; authority decisions read through the same seam"
  - "Confirm/reject authority is any-of over the owner's units (the actor passes when they are the resolved manager of ANY unit membership), documented in the service comment — self-confirm falls out of the same check (D-14-04)"
  - "Withdraw audit carries a nil payload (the port takes no reason — D-14-10); reject carries {reason} (direction.go:439 shape); UpdateMedical's {before, after} payload is built by the REPO from the FOR UPDATE locked row — the service cannot know the before state"
  - "AttachCertificate audit entity_id = the window id (the per-window history contract, Declare precedent); the attachment row gets its own generated id"
  - "Test date assertions use dayMid (UTC midnight) for scanned DATE columns — the shared day() helper builds noon-UTC stamps while PostgreSQL scans DATE at midnight UTC in this session (Phase 13 normalizeDay lesson, direction_repository.go:723)"

patterns-established:
  - "Lifecycle mutators share the Activate skeleton verbatim: lock → re-check → precondition UPDATE → in-tx audit → re-read → commit"
  - "Service gates are fast-fail UX; the repo's in-tx checks are authoritative (CR-01) — the HR edit/attach medical-only check exists at BOTH layers"
  - "Audit DTOs are built service-side with ActorID = the actor; the repo pins entity_id when nil and writes payloads it owns (before/after, attachment)"

requirements-completed: [AVAIL-02]

coverage:
  - id: D1
    description: "Confirm/Reject/Withdraw repo transactions: declared→confirmed/rejected/withdrawn with in-tx audit rows; terminal states (confirmed/rejected/withdrawn) reject every further transition with ErrInvalidTransition; reject reason required at the repo boundary; missing/cross-org ids → ErrWindowNotFound (no existence oracle)"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Confirm"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Reject"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Withdraw"
        status: pass
    human_judgment: false
  - id: D2
    description: "Audit-in-tx atomicity for all five mutators: a failed in-tx audit insert rolls back the status change / medical edit / attachment write (BE-012, T-14g-15)"
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_LifecycleAuditRollback"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_CurationAuditRollback"
        status: pass
    human_judgment: false
  - id: D3
    description: "HR medical edit + certificate attach repo transactions: medical-only in-tx check (ErrNotMedical), {before, after} audit payload (D-14-12), BYTEA round-trip (bytes read back equal bytes written, D-14-07), append-only attachment storage"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_UpdateMedical"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_AttachCertificate"
        status: pass
    human_judgment: false
  - id: D4
    description: "Service confirm/reject authority matrix: resolved unit manager confirms; non-manager / no-manager / hr-not-manager → ErrForbidden with the repo never called (T-14g-04, D-14-01/03); self-confirm allowed (D-14-04)"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_Confirm_AuthorityMatrix"
        status: pass
    human_judgment: false
  - id: D5
    description: "Service reject reason fast-fail (D-14-09) with audit {reason} capture; owner-only withdraw (D-14-10, unit manager is forbidden on others' windows); HR-only edit/attach gates with audit DTO capture (T-14g-13)"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_Reject"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_Withdraw"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/availability_test.go#TestService_HrGates"
        status: pass
    human_judgment: false

# Metrics
duration: 30min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 05: Lifecycle Mutators + Certificate Path (AVAIL-02) Summary

**The AVAIL-02 lifecycle is fully shipped: five repo mutators (Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate) with the CR-01 in-tx matrix re-check under FOR UPDATE, status-precondition backstops, and in-tx audit rollback proofs — plus the service orchestration resolving the confirm/reject authority through the SHARED routing service (unit-manager any-of, self-confirm falls out, HR never confirms), the owner-only withdraw gate, the HR-only medical-edit/certificate-attach gates, and the BYTEA certificate store with append-only attachments.**

## Performance

- **Duration:** 30 min
- **Started:** 2026-08-11T13:44:00Z
- **Completed:** 2026-08-11T14:13:43Z
- **Tasks:** 3 (each a full RED→GREEN TDD cycle)
- **Files modified:** 5

## Accomplishments

- **Repo lifecycle mutators (Task 1):** `Confirm` (declared → confirmed), `Reject` (reason fast-fail at the repo boundary — `ErrRejectReasonRequired`, `rejection_reason` persisted, audit `{reason}`), `Withdraw` (declared → withdrawn, terminal row never hard-deleted). All three follow the Activate skeleton: FOR UPDATE lock → in-tx matrix re-check → UPDATE with the locked status as WHERE precondition → audit in-tx → re-read → commit. Terminal states (rejected/withdrawn) reject every further transition with `ErrInvalidTransition`; missing/cross-org ids → `ErrWindowNotFound` (no existence oracle).
- **HR curation mutators (Task 2):** `UpdateMedical` — medical-only in-tx check (T-14g-13), dates + certificate_ref correction with the `{before, after}` audit payload built from the locked row (D-14-12); `AttachCertificate` — BYTEA row into `certificate_attachments` (content_type/size/storage round-trip proven, D-14-07) with the `certificate_attached` audit row addressed to the window; append-only (second attach lands a second row).
- **Audit atomicity proven twice (BE-012, T-14g-15):** `LifecycleAuditRollback` (confirm + withdraw) and `CurationAuditRollback` (edit + attach) force the audit FK violation (bogus actor → 23503) and assert the state write rolls back with it.
- **Service orchestration (Task 3):** the confirm/reject authority is resolved through the shared routing service — `ListMembershipsForUser` per owner, then `ResolveUnitManager` per unit (any-of), with the HR-never-confirms gate and self-confirm (D-14-04) falling out of the same check (T-14g-04). Withdraw is owner-only (D-14-10); UpdateMedical/AttachCertificate gate on `models.RoleHR` (never a literal). Reject fast-fails the reason before the repo (D-14-09); the medical-only fast-fail mirrors the repo's in-tx authority.
- **Mock upgrade:** `MockUnitRepo.ListMembershipsForUser` now derives memberships from the `UnitMembers` map per the port contract — the availability authority tests seed one map and the routing walk consumes both methods consistently.
- **Service unit batteries:** the full authority matrix (resolved manager / non-manager / self / no-manager / hr-not-manager), reject reason + payload capture, owner-only withdraw, HR gates with audit DTO assertions (action, actor, entity_id) via the mock's `Audits` capture.

## Task Commits

Each task was committed atomically (TDD: test commit then feat commit):

1. **Task 1 RED: repo confirm/reject/withdraw batteries** - `25a88bf` (test)
2. **Task 1 GREEN: implement repo confirm/reject/withdraw** - `443c918` (feat)
3. **Task 2 RED: repo update-medical + attach-certificate batteries** - `c75fc83` (test)
4. **Task 2 GREEN: implement repo update-medical + attach-certificate** - `019436e` (feat)
5. **Task 3 RED: service lifecycle authority batteries** - `3b6ba61` (test)
6. **Task 3 GREEN: implement service lifecycle orchestration** - `e698cf3` (feat)

**Plan metadata:** committed after this file

## Files Created/Modified

- `internal/adapters/secondary/postgres/availability_repository.go` - Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate replacing the 14-03 stubs; all CR-01 txs with in-tx audit (modified)
- `internal/adapters/secondary/postgres/availability_repository_test.go` - transition/terminal/reason/rollback/medical-edit/attach batteries + windowAudit/dayMid helpers (modified)
- `internal/core/services/availability/availability.go` - Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate orchestration + UpdateMedicalRequest/AttachCertificateRequest types + getWindow/unitManagerAuthorized helpers (modified)
- `internal/core/services/availability/availability_test.go` - authority-matrix, reject, withdraw, hr-gate batteries + seedWindow/seedUnitMember/seedUnit/seedManagerChain fixtures (modified)
- `internal/core/services/testdata/mocks.go` - MockUnitRepo.ListMembershipsForUser derives from UnitMembers (modified)

## Decisions Made

- **Window owner/kind resolved via the Windows read** — the pinned port has no single-window getter (14-02 locked the surface), so the service resolves the window through the org-wide `Windows` read (mock-served now, production read lands 14-07). Alternative would have required a port change — rejected.
- **Any-of authority** — the actor passes when they are the resolved unit manager of ANY of the owner's units; the plan's "discretion: any-of, documented in the service comment" honored verbatim.
- **{before, after} built repo-side** — the service cannot know the before state; the repo builds the payload from the FOR UPDATE locked row.
- **Audit entity_id for attachments = the window id** — the per-window history contract (Phase 19 reads filter by window); the attachment row's own id is never the audit address (Declare precedent).
- **dayMid for scanned DATE comparisons** — the shared `day()` helper returns noon-UTC stamps while PostgreSQL scans DATE columns at midnight UTC in this session (Phase 13 normalizeDay lesson); mutator-scan assertions compare against `dayMid`, never `day()`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] MockUnitRepo.ListMembershipsForUser returned nil — the authority resolution could never pass**
- **Found during:** Task 3 (RED authoring)
- **Issue:** The plan's authority resolution consumes `unitRepo.ListMembershipsForUser` (unit_repository.go:33 port method), but the testdata mock hard-coded `return nil, nil` — the service would resolve zero memberships and no actor could ever be authorized, making the authority matrix untestable.
- **Fix:** The mock now derives memberships by scanning the `UnitMembers` map across units (the port contract). Additive — no existing test depended on the nil return; direction/unit tests that seed `UnitMembers` now get consistent memberships.
- **Files modified:** internal/core/services/testdata/mocks.go
- **Verification:** full authority matrix green (manager/self/hr/no-manager/non-manager).
- **Committed in:** 3b6ba61, e698cf3

**2. [Rule 1 - Bug] Test date comparisons failed: shared day() returns noon UTC, scanned DATE columns return midnight UTC**
- **Found during:** Task 2 (GREEN — UpdateMedical assertions failed with a 12:00 vs 00:00 mismatch)
- **Issue:** The `day()` helper (coverage_repository_test.go:458) deliberately builds NOON-UTC stamps, while PostgreSQL scans DATE columns back at MIDNIGHT UTC in this session — naive `.Equal` on scanned dates was nondeterministic per session timezone (the Phase 13 normalizeDay lesson, direction_repository.go:723).
- **Fix:** Added a `dayMid` helper (UTC midnight) for mutator-scan assertions and normalized the scanned side; documented the invariant next to the helper. First patch attempt truncated the expected value instead of the actual — corrected to normalize the scanned side only.
- **Files modified:** internal/adapters/secondary/postgres/availability_repository_test.go
- **Verification:** all four scanned-date assertions green; full postgres package suite green (40s).
- **Committed in:** 019436e

**3. [Rule 1 - Bug] Rejected-row seed violated the 023 2VL CHECK before the reason patch**
- **Found during:** Task 1 (RED — seed insert failed with 23514 on `availability_windows_reject_reason_check`)
- **Issue:** The plan's test wording seeded rejected rows via `seedAvailabilityWindowWithCert` then patched `rejection_reason` with a follow-up UPDATE — but the INSERT itself violates the never-NULL-satisfiable CHECK before the patch runs.
- **Fix:** Rejected rows are inserted inline with the reason in the same statement (the 14-03 active-only battery precedent).
- **Files modified:** internal/adapters/secondary/postgres/availability_repository_test.go
- **Verification:** reject-terminal subtests green.
- **Committed in:** 25a88bf

---

**Total deviations:** 3 auto-fixed (3 Rule 1 bugs in test infrastructure/assertions — no production logic changes beyond the plan's spec)
**Impact on plan:** All three fixes were required to make the tests express the plan's behavior truthfully. No scope creep — the mutator and service implementations follow the plan's skeleton verbatim.

## Issues Encountered

- **Stale patch application:** one Python string-replace pass silently didn't match (whitespace drift after gofmt), leaving a half-applied date fix — caught by the repeated failing assertion, fixed with targeted Edit calls. Lesson: verify replacements with grep after scripted edits.
- **`normalizeDay` redeclaration:** the package already exports `normalizeDay` (direction_repository.go:723) — my local helper collided; removed in favor of `dayMid` (different purpose: expected-value side).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-06 (contract-type CRUD):** the remaining repo stubs (ListContractTypes/Create/Update/Delete) are untouched by this plan — clean RED surface for the next plan.
- **14-07 (read-models):** the service's `getWindow` authority resolution currently rides the mock's `Windows` stub — 14-07's production `Windows` read is a dependency of the lifecycle routes (noted in the service comment).
- **14-08 (HTTP surface):** the handler battery exercises the full lifecycle over the fixture; the service method signatures match the 14-PATTERNS handler shape (orgID, actorID, role, id, …).
- **Phase 19 history filters:** the audit vocabulary (confirmed/rejected/withdrawn/edited/certificate_attached) is pinned in the domain; entity_id = window for every lifecycle event.
- Full suite green (`make test` exit 0, 26 packages).

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*

## Self-Check: PASSED

- All 5 task files exist on disk (availability_repository.go/.test.go, availability.go/.test.go, testdata/mocks.go).
- All 6 plan commits verified in git history (25a88bf, 443c918, c75fc83, 019436e, 3b6ba61, e698cf3).
- Plan-level verification re-run: `go test ./internal/adapters/secondary/postgres/ -run TestAvailabilityRepository` ok; `go test ./internal/core/services/availability/` ok; `go build ./...` ok; `make test` full-suite exit 0 (26 packages).
- TDD gate compliance: 3 RED (`test(14-05)`) + 3 GREEN (`feat(14-05)`) commits, each RED immediately preceding its GREEN.
