---
phase: 09-activity-ontology
plan: 04
subsystem: api
tags: [go, hexagonal, services, approval-routing, adr-be-014, adr-p-007]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: 011 migration + ActivityRepository port/adapter with GetAncestry / ResolveCommercialContext / ResolveBillability (plans 01 + 03)
provides:
  - ActivityService (full CRUD + D-2 kind-catalog / D-3 contract validation, sentinel-guarded Delete) replacing the collapsed project service
  - Time-entry + expense approval routing on the activity chain (R-1/R-2/R-3): WG manager/delegate, unit-manager fallback, ErrActivityNotLoggable, D-11 skip incl. delegates
  - Approve-side approver-set verification at the manager stage (WG/unit routing enforceable at approval time)
  - ports.ActivityRepository.KindExists + shared ErrActivityNotLoggable sentinel in the activity domain
affects: [09-05 http handlers, phase 10 frontend rename, working_group renovation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Routing precedence chain: entry → activity → anchored WG → manager+delegates, unit-manager fallback (upward tree walk) for personal activities, ErrActivityNotLoggable for commercial-no-WG — identical in both entry services"
    - "Role-based D-11 skip: owner ∈ approver set (manager OR delegate) → straight to pending_finance; member-only does not fire"
    - "Manager stage is a resolved approver set (not a role label): Approve re-resolves the set and verifies membership; terminal unit-tree case stays role-gated per ADR-BE-014 consequences"

key-files:
  created:
    - internal/core/services/activity/activity.go
    - internal/core/services/activity/activity_test.go
  modified:
    - internal/core/services/time_entry/time_entry.go
    - internal/core/services/time_entry/time_entry_test.go
    - internal/core/services/expense/expense.go
    - internal/core/services/expense/expense_test.go
    - internal/core/ports/activity_repository.go
    - internal/adapters/secondary/postgres/activity_repository.go
    - internal/core/domain/activity/activity.go
    - internal/core/services/testdata/mocks.go
    - internal/core/services/testdata/mocks_test.go
    - internal/core/services/testdata/factories.go
  deleted:
    - internal/core/services/project/project.go
    - internal/core/services/project/project_test.go
    - internal/core/services/project/project_integration_test.go

key-decisions:
  - "ErrActivityNotLoggable sentinel lives in the activity domain (single source shared by both entry services, ADR-BE-001)"
  - "Manager-stage approver set = WG row ManagerID + DelegateIDs (R-1 routing source), not wg_members roles; the unit-manager fallback walks unit_memberships.role='manager' up the tree"
  - "Approve re-resolves the approver set and verifies membership — makes R-1/R-2 routing enforceable at approval time, not just at submit"
  - "Terminal unit-tree case (org root without manager) stays role-gated per ADR-BE-014 consequences ('routes to the org-role manager') — the service cannot pin a user there"
  - "UnitRepository wired into ActivityService per the plan's dependency list (R-4 visibility); entry-level visibility lives repo-side, so it is reserved, not dead-gated"
  - "working_group_integration_test.go left red (pre-existing project seeding, out of plan scope) — logged to deferred-items.md"

patterns-established:
  - "Both entry services share the identical resolveManagerStage/resolveUnitManager chain — mirrored per the plan's 'pattern to mirror' instruction, with the same acceptance matrix enforcing identity"

requirements-completed: [BE-014-R1, BE-014-R2, BE-014-R3, BE-014-R4, P-007-D8]

# Metrics
duration: 7min
completed: 2026-07-31
---

# Phase 09 Plan 04: Service Layer — Routing Rewrite + Activity Service Summary

**Approval routing moves from pinned FKs to the activity chain end-to-end: a new ActivityService (CRUD + D-2/D-3 validation + sentinel-guarded Delete) replaces the collapsed project service, and both time-entry and expense services now resolve the manager stage through activity → anchored WG → manager/delegate with the unit-manager fallback for personal activities (R-2), ErrActivityNotLoggable enforcement for commercial-no-WG (409), and the D-11 skip extended to delegates (R-3) — with Approve verifying the actor against the re-resolved approver set so routing is enforceable at approval time, not just submission.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T16:39:21Z
- **Completed:** 2026-07-31T16:46:25Z
- **Tasks:** 3
- **Files modified:** 16 (2 created, 11 modified, 3 deleted)

## Accomplishments

- `ActivityService` with the full plan surface: Create (validates kind against the org's `activity_kinds` catalog per D-2 via the new `KindExists` port method, parent same-org, contract existence per D-3), GetByID, Update/Delete (finance-role gating + created-by-org owner check, Delete returning clean sentinels for has-children / active entries per ADR-BE-001), List by org/contract/parent, ListChildren, Adopt, manager management — replacing `services/project` entirely (R-6)
- Time-entry `Submit` rewritten per ADR-BE-014: R-1 (activity → anchored WG → WG manager + delegates), R-2 fallback (personal activities → submitter's unit manager via `unit_memberships.role='manager'`, walking the unit tree upward), R-2 enforcement (commercial activity without an anchored WG → `ErrActivityNotLoggable`), R-3 D-11 skip (owner in the approver set — manager OR delegate — goes straight to `pending_finance`; being a mere WG member does not fire it)
- Time-entry `Approve` now re-resolves the manager-stage approver set and verifies the actor's membership — a random user can no longer act as "manager"; the finance stage is unchanged; self-approval is structurally impossible (owner gate + D-11 skip + set membership)
- Expense service routing is identical to time entries (ADR-P-001 Q1, unblocked by the shared `activity_id` FK): same chain, same fallback, same D-11 skip, same enforcement — and no project-manager/activity-manager approval queue remains anywhere (R-2 note)
- R-4 visibility pass-through verified: `ListPending` delegates to the repositories whose unit-subtree gating landed in Plan 03 — no second filter added
- Full service unit test matrix for both entry types (WG routing, unit-manager fallback + upward walk, ErrActivityNotLoggable, D-11 skip manager/delegate, D-11 no-fire member, approver-set verification, self-approval impossible) plus the activity service test suite

## Task Commits

Each task was committed atomically:

1. **Task 1: ActivityService + KindExists port + delete project service** - `fd4124f` (feat)
2. **Task 2: Time-entry routing rewrite (R-1/R-2/R-3/R-4)** - `e508fee` (feat)
3. **Task 3: Expense routing rewrite (identical chain)** - `eb32935` (feat)

**Plan metadata:** pending docs commit

## Self-Check: PASSED

- All created files exist on disk (activity.go, activity_test.go, 09-04-SUMMARY.md)
- All 3 task commits present in git log: `fd4124f`, `e508fee`, `eb32935`
- `go test ./internal/core/services/...` — 11/12 packages PASS (sole failure: pre-existing `working_group_integration_test.go` seeding dropped `projects` tables — logged to deferred-items.md, out of scope)
- `go build ./internal/core/...` — PASS
- `go vet` on services + ports + activity domain — clean

## Files Created/Modified

- `internal/core/services/activity/activity.go` - ActivityService: Create (kind-catalog/parent-org/contract validation), GetByID, List, ListChildren, Update/Delete (finance gate + owner check + sentinel guards), Adopt, manager management; UnitRepository wired per plan (R-4 visibility, reserved)
- `internal/core/services/activity/activity_test.go` - 19 sub-tests: Create validation matrix, Update/Delete role gating + owner check, guard sentinels, List/ListChildren, managers, Adopt
- `internal/core/services/time_entry/time_entry.go` - Submit/Approve rewritten on the activity chain; resolveManagerStage/resolveUnitManager; Approve-set verification; Create/Update on ActivityID+UnitID; ListPending pass-through
- `internal/core/services/time_entry/time_entry_test.go` - full routing matrix + preserved non-routing tests
- `internal/core/services/expense/expense.go` - identical routing chain (mirrored); no project-manager queue; Create/Update on ActivityID
- `internal/core/services/expense/expense_test.go` - mirrored routing matrix + preserved non-routing tests
- `internal/core/ports/activity_repository.go` - +KindExists (D-2 catalog check for Create validation)
- `internal/adapters/secondary/postgres/activity_repository.go` - KindExists implementation (EXISTS against activity_kinds)
- `internal/core/domain/activity/activity.go` - +ErrActivityNotLoggable sentinel (shared by both entry services)
- `internal/core/services/testdata/mocks.go` - MockProjectRepo → MockActivityRepo (with ResolveCommercialContext derivation + KindExists + guard Fns); MockWorkingGroupRepo.ListByOrg now honors org + activity filter
- `internal/core/services/testdata/mocks_test.go` - instantiation list uses MockActivityRepo
- `internal/core/services/testdata/factories.go` - NewProject → NewActivity; NewExpenseDomain ProjectID → ActivityID
- Deleted `internal/core/services/project/` (project.go, project_test.go, project_integration_test.go)
- `.planning/phases/09-activity-ontology/deferred-items.md` - logged pre-existing working_group integration breakage

## Decisions Made

- **ErrActivityNotLoggable lives in the activity domain** — one sentinel shared by both entry services (the plan says "same ErrActivityNotLoggable enforcement"); both services return it and 09-05's handlers will `errors.Is`-match it to 409.
- **Manager-stage approver set = WG row `ManagerID` + `DelegateIDs`** — the R-1 routing source, matching the plan's "WG manager + delegates" wording (not `wg_members` roles, which are a separate membership concern).
- **Approve re-resolves the approver set and verifies membership** — without this, the routing rewrite would be a no-op at approval time (any non-owner could pass role="manager"). The pre-rewrite `Approve` only checked owner ≠ actor. Classified as Rule 2 (missing critical functionality for the routing rewrite's own objective).
- **Terminal unit-tree case stays role-gated** — when no unit manager exists up to the org root, ADR-BE-014's consequence ("routes to the org-role manager") means the service cannot pin a user; the handler's role resolution governs. Documented in `managerResolution.roleGated`.
- **UnitRepository wired into ActivityService per the plan's dependency list** — the plan names it "for visibility"; entry-level visibility (R-4) lives in the entry repositories from Plan 03, so the dependency is carried in the constructor and reserved rather than gating (activity List is org-scoped via the filter).
- **working_group integration test deferred** — pre-existing breakage (seeds tables dropped by migration 011), root cause predates this plan; the working_group service is absent from plans 04/05 file lists. Logged to deferred-items.md with a suggested fix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `KindExists` to the ActivityRepository port**
- **Found during:** Task 1 (activity service Create design)
- **Issue:** The plan requires Create to "validate kind exists in the org's activity_kinds catalog (D-2)", but the Plan-03 port has no way to check catalog membership — without it, an unknown kind surfaces as an ambiguous FK violation (indistinguishable from a bad parent/contract FK).
- **Fix:** Added `KindExists(ctx, orgID, kind) (bool, error)` to `ports.ActivityRepository`, implemented it in the postgres adapter (`EXISTS` against `activity_kinds`), and in the test mock. Create now returns `ErrInvalidRequest` for unknown kinds.
- **Files modified:** internal/core/ports/activity_repository.go, internal/adapters/secondary/postgres/activity_repository.go, internal/core/services/testdata/mocks.go
- **Verification:** activity service tests ("kind not in org catalog rejected (D-2)") pass; postgres activity-repo tests still pass.
- **Committed in:** fd4124f (Task 1 commit)

**2. [Rule 2 - Missing Critical] `ErrActivityNotLoggable` sentinel added to the activity domain**
- **Found during:** Task 1 (sentinel placement)
- **Issue:** The plan references `ErrActivityNotLoggable` as the shared sentinel both entry services must return (must_haves), but no such error was defined anywhere.
- **Fix:** Declared it in `internal/core/domain/activity/activity.go` (single source for both services, per ADR-BE-001 sentinel conventions).
- **Files modified:** internal/core/domain/activity/activity.go
- **Verification:** both services return it via `errors.Is`; tests assert `activitydomain.ErrActivityNotLoggable`.
- **Committed in:** fd4124f (Task 1 commit)

**3. [Rule 3 - Blocking] testdata mocks/factories still referenced the deleted project domain**
- **Found during:** Task 1 (testdata package compile)
- **Issue:** `mocks.go`/`factories.go`/`mocks_test.go` imported `domain/project` (deleted in Plan 03) — the whole testdata package was uncompilable, blocking every services test package. `NewExpenseDomain` also still set the old `ProjectID` field.
- **Fix:** Replaced `MockProjectRepo` with `MockActivityRepo` (implementing the full ActivityRepository port incl. KindExists + configurable guard Fns + derived ResolveCommercialContext); `NewProject` → `NewActivity`; `ProjectID` → `ActivityID` in NewExpenseDomain.
- **Files modified:** internal/core/services/testdata/mocks.go, factories.go, mocks_test.go
- **Verification:** `go vet ./internal/core/services/testdata/` clean; instantiation test passes.
- **Committed in:** fd4124f (Task 1 commit)

**4. [Rule 3 - Blocking] `MockWorkingGroupRepo.ListByOrg` ignored the activity filter**
- **Found during:** Task 1 (routing design review)
- **Issue:** The mock returned every WG regardless of org or anchored activity. The R-1 chain resolves the manager stage via `ListByOrg(ctx, orgID, &activityID)` — a non-filtering mock would make every routing test pass vacuously (first WG in map wins).
- **Fix:** Updated the mock to filter by orgID and the anchored activity (the legacy `SubprojectID` field, which maps to `activities.activity_id`).
- **Files modified:** internal/core/services/testdata/mocks.go
- **Verification:** routing tests distinguish "activity with anchored WG" from "activity without one" (R-2 fallback / enforcement tests pass).
- **Committed in:** fd4124f (Task 1 commit)

**5. [Rule 2 - Missing Critical] Approve-side approver-set verification at the manager stage**
- **Found during:** Task 2 (routing rewrite design)
- **Issue:** The pre-rewrite `Approve` only checked owner ≠ actor and the role label — anyone could pass role="manager" and approve any submitted entry. The plan's R-1/R-2 define the manager stage as a resolved approver set (WG manager/delegate or unit manager); without a membership check at approval time, the routing would only affect the status label, not who can approve.
- **Fix:** `Approve` re-resolves the manager stage and verifies the actor is in the approver set (`!roleGated && !contains(approverIDs, userID) → ErrForbidden`). Combined with the owner gate and the D-11 skip, self-approval is structurally impossible and non-approvers are rejected.
- **Files modified:** internal/core/services/time_entry/time_entry.go, internal/core/services/expense/expense.go
- **Verification:** "non-approver cannot act at the manager stage" tests pass for both entry types.
- **Committed in:** e508fee (Task 2), eb32935 (Task 3)

---

**Total deviations:** 5 auto-fixed (2 missing-critical ×2, 2 blocking, 1 missing-critical routing enforcement)
**Impact on plan:** All fixes were required for the plan's own acceptance criteria to be implementable (kind validation, shared sentinel), for the test infrastructure to compile against the Plan-03 collapse, or for the routing rewrite to be enforceable rather than cosmetic. No scope creep — no handler, router, or migration changes.

## Issues Encountered

- **Expected mid-phase breakage (not a deviation):** `cmd/server/main.go`, `cmd/server/main_test.go`, and `internal/adapters/primary/http/{project.go, project_test.go, handler_test_helper.go}` still reference the deleted project domain/service packages and do not compile. This is the accepted mid-phase state documented in the 09-03 summary — 09-05 (HTTP Handlers + Route Wiring) rewires them.
- **Pre-existing red (out of scope):** `working_group_integration_test.go` seeds the `projects`/`subprojects` tables dropped by migration 011 → runtime-red since 09-01. It is the sole failing package in `go test ./internal/core/services/...`. Logged to deferred-items.md with a suggested fix (re-seed with activities; belongs with the working_group renovation).

## Verification Results

- `go test ./internal/core/services/...` — **11/12 PASS** (activity, auth, contract incl. integration, customer, expense, export, invitation, organization, password_reset, time_entry, unit green; only pre-existing working_group integration red)
- `go build ./internal/core/...` — **PASS** (services compile against the new ports)
- `go vet ./internal/core/services/... ./internal/core/ports/ ./internal/core/domain/activity/` — **PASS**
- `go test ./internal/adapters/secondary/postgres/ -run 'TestActivity'` — **PASS** (KindExists port addition regressions nothing)
- Coverage confirmed: R-1 chain (WG routing test), R-2 fallback + upward walk + enforcement (3 tests), R-3 skip incl. delegates (3 tests), R-4 visibility pass-through (both entry types)
- `requirements mark-complete BE-014-R1 BE-014-R2 BE-014-R3 BE-014-R4 P-007-D8` — **not_found**: the plan's `requirements:` frontmatter references ADR decision IDs (ADR-BE-014 R-1…R-4, ADR-P-007 D-8) that are not registered in `.planning/REQUIREMENTS.md` (which tracks milestone-level IDs like TEST-*/AUTH-*). Same situation as plans 09-01…09-03, whose docs commits also left REQUIREMENTS.md untouched. The SUMMARY's `requirements-completed` frontmatter is still populated verbatim per the template contract; the ADR decisions themselves are tracked in `hourglass-vault/decisions/`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Service layer fully on the activity ontology: ActivityService replaces project service; both entry services route through activity → WG → manager/delegate with unit-manager fallback, D-11 skip incl. delegates, and ErrActivityNotLoggable enforcement
- **Ready for 09-05 (HTTP Handlers + Route Wiring)** — consumes `services/activity` (CRUD surface), the new `time_entry`/`expense` service signatures (NewService now takes wg/activity/unit repos), and must map `ErrActivityNotLoggable` → 409 (its Task 2 acceptance)
- **Heads-up for the 09-05 executor:** the entry service constructors changed (5-dependency / 4-dependency); `cmd/server/main.go` must wire the new repos. `handler_test_helper.go` uses `services/project` mocks — rewrite against `services/activity`. The activity handler's detail endpoint composes `GetAncestry`/`ResolveCommercialContext`/`ResolveBillability` from the repo — the service exposes the CRUD surface per this plan's scope; a `ListKinds` port/service method for `GET /api/activity-kinds` is a trivial addition when 09-05 needs it.
- **Blockers/concerns:** none from this plan. The pre-existing working_group integration breakage and the 000 down-migration gap remain tracked in deferred-items.md.

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*
