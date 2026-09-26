---
phase: 14-availability-backend-absences-capacity
plan: 06
subsystem: backend
tags: [go, postgres, contract-types, work-schedule, resolve-schedule, fallback-chain, hr-gate, audit, D-14-16..29]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity (plan 01)
    provides: migration 024 contract_types + organization_memberships.contract_type_id/day_hours_override; seedContractType helper
  - phase: 14-availability-backend-absences-capacity (plan 02)
    provides: the pinned ContractType/ScheduleResolution/EmployeeSchedule domain types, DayHoursValid helper, ErrContractTypeNotFound/ErrContractTypeInUse sentinels, ports.AvailabilityRepository signatures
  - phase: 14-availability-backend-absences-capacity (plan 03)
    provides: the *AvailabilityRepository type + insertAvailabilityAudit helper + the not-implemented contract-type stubs the CRUD replaces; the shared orgsettings/routing service fixture
  - phase: 13-direction-backend-the-plan-plane
    provides: the org_settings key/value store + orgsettingssvc.Service (D-13-18, the org-default key seam)
provides:
  - Contract-type repo CRUD (List/Create/Update/Delete on *AvailabilityRepository) with in-tx audit (created/updated/deleted) and the FK-safe hard delete (23503 → ErrContractTypeInUse → 409, D-14-28)
  - The service layer: code-side JSONB validation (T-14g-18), hr-gated CRUD with audit DTOs (T-14g-16, D-14-27), and ResolveSchedule — the per-employee fallback chain override → type → org default → 8×5 with exported resolution levels (D-14-18) and invalid-default surfacing (ErrInvalidValue, T-14g-19)
  - The membership schedule endpoint PUT /organizations/members/{member_id}/schedule (D-14-29): hr-gated, same-org type validation via the SHARED availability service, {before, after} audit in-tx
  - The org-default key path: default_contract_type_id added to the orgsettings known-key vocabulary (the Phase 13 endpoint now stores it), read by ResolveSchedule
  - Wiring: 5 new routes in cmd/server/main.go + the fixture; orgsvc.NewService gained the shared availability service (D-G parity) at all 5 call sites; postgres GetMembership/scanMemberships surface the 024 columns
affects: [14-07 capacity read-model (consumes ResolveSchedule per employee + resolution level), 14-08 HTTP surface (route battery over the lifecycle + contract types), Phase 19 history filters (audit vocabulary: contract_type created/updated/deleted, organization_membership schedule_updated)]

actuals:
  tokens: 27122    # chars/4 over the realized diff (108491 chars, 23 files, 7 commits) — estimate was 20000
  tasks: 3
  commits: 7

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Contract-type mutators share the Declare skeleton: BeginTx → FOR UPDATE lock (ErrNoRows → ErrContractTypeNotFound, no existence oracle) → mutate → audit in-tx → re-read → commit"
    - "The FK-in-use delete guard: a 23503 on DELETE contract_types IS the in-use guard (the only FK is organization_memberships.contract_type_id) — wrapPGError → ports.ErrForeignKey → ErrContractTypeInUse"
    - "ResolveSchedule fallback chain with the resolution level returned in the response (Pattern 4): membership override merge (override wins per weekday, 8×5 base without a type) → membership type → org default key → 8×5 fallback"
    - "Vocabulary-first: the audit constants (contract_type created/updated/deleted, organization_membership schedule_updated) + ErrInvalidValue pinned in the availability domain BEFORE code landed (Phase 13 precedent)"
    - "The org service reuses the SHARED availability service for same-org validation — no second repo, no second instance (D-G parity)"

key-files:
  created:
    - internal/adapters/secondary/postgres/contract_type_repository.go
    - internal/adapters/secondary/postgres/contract_type_repository_test.go
    - internal/core/services/availability/contract_types.go
    - internal/core/services/availability/contract_types_test.go
    - internal/adapters/primary/http/contract_types_handler.go
    - internal/adapters/primary/http/contract_types_handler_test.go
  modified:
    - internal/adapters/secondary/postgres/availability_repository.go
    - internal/adapters/secondary/postgres/organization_repo.go
    - internal/adapters/secondary/postgres/organization_membership_repo.go
    - internal/adapters/secondary/postgres/organization_management_repo.go
    - internal/core/ports/organization_management_repository.go
    - internal/core/services/organization/organization.go
    - internal/core/services/organization/organization_test.go
    - internal/core/services/organization/organization_integration_test.go
    - internal/core/services/testdata/mocks.go
    - internal/core/domain/availability/availability.go
    - internal/core/domain/availability/errors.go
    - internal/core/domain/auth/membership.go
    - internal/core/domain/orgsettings/orgsettings.go
    - internal/adapters/primary/http/organization.go
    - internal/adapters/primary/http/availability_handler.go
    - internal/adapters/primary/http/handler_test_helper.go
    - cmd/server/main.go
    - cmd/server/main_test.go

key-decisions:
  - "The orgsvc constructor's 'audit writer parameter' is the SHARED availability service: it validates contract_type_id same-org via ListContractTypes (no second repo) and pins the schedule audit vocabulary (D-14-29, D-G parity)"
  - "The {before, after} audit payloads are built REPO-side from the FOR UPDATE locked rows (contract-type update + membership schedule — the UpdateMedical shape, 14-05): the service cannot know the before state"
  - "ScheduleRequest fields are optional but at least one must be present (no-op writes → 400); a cross-org/missing contract_type_id → 400 invalid request, not 404 (the plan's 404/400 discretion resolved to 400 — it is a value error, and the route asserts it)"
  - "Membership audit entity_id = the membership row id (the {member_id} path param) — the per-membership history contract"
  - "DELETE /availability/contract-types returns 200 with {id} (the plan's integration battery pins 200, not 204)"
  - "default_contract_type_id is a write-time-UUID-string + read-time-existence/org-validated key: the orgsettings validator only checks addressability (T-13-11 JSONB), ResolveSchedule surfaces missing/wrong-org/unparsable as ErrInvalidValue (T-14g-19)"
  - "An override WITHOUT a contract type merges over the 8×5 fallback base (the flagged-assumption discretion pinned with a test)"

patterns-established:
  - "Service gates are fast-fail UX; the repo's in-tx checks are authoritative — the schedule service validates same-org via the shared availability service, the repo locks + audits in-tx (BE-016 Pitfall 2)"
  - "Contract-type validation uses the 14-02 domain helper (DayHoursValid) — no re-implementation drift"
  - "The org service is constructed AFTER the availability service in every wiring site (main.go, fixture, main_test.go, org integration test) — the schedule endpoint's dependency order"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "Contract-type repo CRUD: org-scoped list, create/update/delete with in-tx audit rows (created/updated/deleted), FK-in-use delete → ErrContractTypeInUse (D-14-28), missing/cross-org → ErrContractTypeNotFound (no existence oracle)"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/contract_type_repository_test.go#TestContractTypeRepository_ListAndCreate"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/contract_type_repository_test.go#TestContractTypeRepository_Update"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/contract_type_repository_test.go#TestContractTypeRepository_Delete"
        status: pass
    human_judgment: false
  - id: D2
    description: "Service contract-type layer: code-side JSONB validation (T-14g-18, zero repo calls on invalid payloads), hr write gate with manager/employee read-only (T-14g-16, D-14-27), audit DTO capture (created/updated/deleted) via the mock"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/core/services/availability/contract_types_test.go#TestContractTypes_Validation"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/contract_types_test.go#TestContractTypes_HrGates"
        status: pass
    human_judgment: false
  - id: D3
    description: "ResolveSchedule fallback chain (D-14-18): override merge (wins per weekday, 8×5 base without a type) → membership type → org default key → 8×5 fallback, resolution levels returned (Pattern 4); invalid stored defaults surface ErrInvalidValue (T-14g-19)"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/core/services/availability/contract_types_test.go#TestResolveSchedule"
        status: pass
    human_judgment: false
  - id: D4
    description: "Membership schedule endpoint PUT /organizations/members/{member_id}/schedule (D-14-29): hr 200 + membership row updated + schedule_updated audit in-tx; non-hr 403 (T-14g-17); cross-org type id → 400; contract-type routes wired (CRUD permission matrix, FK delete → 409)"
    requirement: AVAIL-02
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/contract_types_handler_test.go#TestContractTypesHandler"
        status: pass
    human_judgment: false
  - id: D5
    description: "Org default schedule key: PUT /organizations/settings default_contract_type_id stores (D-13-18 vocabulary extended), consumed by ResolveSchedule (service-level assertion in D3)"
    requirement: AVAIL-02
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/contract_types_handler_test.go#TestContractTypesHandler/org_default_key_stores_via_PUT_/organizations/settings_(D-14-18)"
        status: pass
    human_judgment: false

# Metrics
duration: 31min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 06: Contract-Type CRUD + ResolveSchedule Fallback Chain (AVAIL-01/02) Summary

**The work-schedule model is fully live: contract-type CRUD (repo + service + routes, hr-owned with FK-safe hard delete), the per-employee ResolveSchedule fallback chain (override → type → org default → 8×5) with resolution levels and invalid-default surfacing, and the hr-gated membership schedule endpoint with in-tx {before, after} auditing — plus the default_contract_type_id org key now servable by the Phase 13 settings endpoint.**

## Performance

- **Duration:** 31 min
- **Started:** 2026-08-11T14:25:27Z
- **Completed:** 2026-08-11T14:56:37Z
- **Tasks:** 3 (each a full RED→GREEN TDD cycle)
- **Files modified:** 23 (6 created, 17 modified)

## Accomplishments

- **Contract-type repo CRUD (Task 1):** `ListContractTypes` (org-scoped), `CreateContractType`, `UpdateContractType`, `DeleteContractType` on `*AvailabilityRepository`, replacing the 14-03 stubs. All mutators follow the Declare skeleton: BeginTx → FOR UPDATE lock (missing/cross-org → `ErrContractTypeNotFound`, no existence oracle) → mutate → audit row IN the same tx (BE-012) → re-read → commit. `Update` writes the `{before, after}` payload from the locked row; `Delete` maps the 23503 FK violation (the only reference is `organization_memberships.contract_type_id`) to `ErrContractTypeInUse` → 409 (D-14-28, T-14g-20) — never a 500. `day_hours` JSONB round-trips `map[string]float64`.
- **Service layer + ResolveSchedule (Task 2):** code-side JSONB validation via the 14-02 `DayHoursValid` helper (cadence vocab, hours > 0, keys "1".."7", 0 < h ≤ 24, week-cadence matrix requirement — T-14g-18), the hr write gate (`models.RoleHR`, T-14g-16) with created/updated/deleted audit DTOs, and the D-14-18 fallback chain: membership `contract_type_id` + `day_hours_override` (override merges over the type matrix — override wins per weekday; an override WITHOUT a type merges over the 8×5 fallback base, the flagged-assumption discretion pinned with a test) → org `default_contract_type_id` key (via the SHARED orgsettings service) → 8h × Mon–Fri. Every resolution returns its level (`override`/`type`/`org_default`/`fallback`, Pattern 4); invalid stored defaults surface `ErrInvalidValue` (T-14g-19, ResolvePlanningMode precedent).
- **Membership schedule endpoint + wiring (Task 3):** `PUT /organizations/members/{member_id}/schedule` — hr-gated (T-14g-17), request `{contract_type_id, day_hours_override}` (optional-but-at-least-one), same-org type validation through the SHARED availability service (`ListContractTypes`), repo `UpdateMembershipSchedule` with the `{before, after}` audit row (`entity_type 'organization_membership'`, action `schedule_updated`) in-tx (BE-016 Pitfall 2). Contract-type routes `GET/POST /availability/contract-types` + `PUT/DELETE /availability/contract-types/{id}` on the `AvailabilityHandler` (D-14-25 discretion). All five new routes mirrored in `cmd/server/main.go` and the fixture; `orgsvc.NewService` gained the shared availability service at all **five** call sites (the plan counted three — `organization_integration_test.go` and `organization_test.go` also construct it, compile-forced).
- **Org-default key path:** `default_contract_type_id` added to the orgsettings known-key vocabulary with a UUID-string validator — the Phase 13 `PUT /organizations/settings` endpoint now stores it (the plan assumed it was already served; the closed `knownKeys` map would have rejected it with `ErrUnknownKey`). ResolveSchedule reads it through the shared orgsettings service; missing/wrong-org/unparsable → `ErrInvalidValue`.
- **Production read path completed:** postgres `GetMembership` and `scanMemberships` now surface the 024 `contract_type_id` + `day_hours_override` columns (JSONB round-trip) — without this, production `ResolveSchedule` would never see membership overrides.
- **Domain pins:** audit vocabulary for the work-schedule plane (`contract_type` created/updated/deleted, `organization_membership` schedule_updated) and `availability.ErrInvalidValue` pinned vocab-first in the domain (Phase 13 precedent); `ScheduleResolution` gained `ContractTypeID` (plan-mandated, additive); `auth.OrganizationMembership` gained the 024 fields.

## Task Commits

Each task was committed atomically (TDD: test commit then feat commit):

1. **Task 1 RED: contract-type repo batteries** - `23a015d` (test)
2. **Task 1 GREEN: implement contract-type repo CRUD** - `824f66e` (feat)
3. **Task 2 RED: service validation + ResolveSchedule batteries** - `96b356c` (test)
4. **Task 2 GREEN: implement service + fallback chain** - `239da0f` (feat)
5. **Task 3 RED: integration battery (routes absent)** - `f9d95f2` (test)
6. **Task 3 GREEN: wire routes + schedule endpoint** - `1cee434` (feat)
7. **gofmt alignment in availability errors block** - `9147cea` (style)

**Plan metadata:** committed after this file

## Files Created/Modified

- `internal/adapters/secondary/postgres/contract_type_repository.go` - List/Create/Update/DeleteContractType on *AvailabilityRepository (audit in-tx, 23503 → ErrContractTypeInUse) (created)
- `internal/adapters/secondary/postgres/contract_type_repository_test.go` - CRUD + FK-in-use + not-found batteries (created)
- `internal/core/services/availability/contract_types.go` - validation, hr-gated CRUD, ResolveSchedule with exported level constants (created)
- `internal/core/services/availability/contract_types_test.go` - validation table, hr gates, fallback-chain batteries (created)
- `internal/adapters/primary/http/contract_types_handler.go` - contract-type routes on *AvailabilityHandler (created)
- `internal/adapters/primary/http/contract_types_handler_test.go` - full integration battery (created)
- `internal/adapters/secondary/postgres/availability_repository.go` - CRUD replaces the 14-03 stubs (modified)
- `internal/adapters/secondary/postgres/organization_repo.go` + `organization_membership_repo.go` - 024 columns surfaced in membership scans (modified)
- `internal/adapters/secondary/postgres/organization_management_repo.go` - UpdateMembershipSchedule with in-tx {before, after} audit (modified)
- `internal/core/ports/organization_management_repository.go` - UpdateMembershipSchedule port method (modified)
- `internal/core/services/organization/organization.go` - availabilitySvc dependency + UpdateMembershipSchedule (modified)
- `internal/core/services/organization/organization_test.go` + `organization_integration_test.go` - constructor call sites (modified)
- `internal/core/services/testdata/mocks.go` - MockOrgMgmtRepo implements the port (schedule write capture) (modified)
- `internal/core/domain/availability/availability.go` + `errors.go` - schedule audit vocabulary, ErrInvalidValue, ScheduleResolution.ContractTypeID (modified)
- `internal/core/domain/auth/membership.go` - ContractTypeID + DayHoursOverride fields (modified)
- `internal/core/domain/orgsettings/orgsettings.go` - KeyDefaultContractTypeID + validator (modified)
- `internal/adapters/primary/http/organization.go` + `availability_handler.go` + `handler_test_helper.go` - schedule handler, writeError sentinels, fixture wiring (modified)
- `cmd/server/main.go` + `main_test.go` - routes + constructor call sites (modified)

## Decisions Made

- **The constructor's "audit writer parameter" is the SHARED availability service** — it validates `contract_type_id` same-org via `ListContractTypes` (no second repo) and pins the schedule audit vocabulary; the plan's three-call-site count was actually five (integration + mock tests also construct orgsvc).
- **{before, after} payloads built repo-side** from the FOR UPDATE locked rows — the service cannot know the before state (the UpdateMedical shape).
- **ScheduleRequest fields optional-but-at-least-one** — a no-op write is `ErrInvalidRequest` (400); a cross-org/missing type is also 400 (a value error, the plan's 404/400 discretion resolved to 400, asserted by the battery).
- **Membership audit entity_id = the membership row id** (the `{member_id}` path param) — the per-membership history contract.
- **DELETE contract-type returns 200 with {id}** — the plan's integration battery pins 200.
- **default_contract_type_id: write-time UUID-string validator + read-time existence/org validation** — the orgsettings validator only guarantees addressability (T-13-11); ResolveSchedule surfaces missing/wrong-org/unparsable as `ErrInvalidValue`.
- **Override-without-type merges over the 8×5 base** — the flagged-assumption discretion (D-14-16 most-complex case) pinned with an explicit test.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] `default_contract_type_id` added to the orgsettings known-key vocabulary**
- **Found during:** Task 3 planning (before RED authoring)
- **Issue:** The plan assumed `PUT /organizations/settings` with key `default_contract_type_id` was "already served by the Phase 13 orgsettings endpoint" — but the orgsettings domain's closed `knownKeys` map rejects unknown keys with `ErrUnknownKey` → 400. Without the vocabulary addition the org-default path (a plan MUST-HAVE) was impossible.
- **Fix:** Added `orgsettings.KeyDefaultContractTypeID` + a UUID-string validator to the domain vocabulary (additive; no closed-vocab test asserted the key set, verified before changing).
- **Files modified:** internal/core/domain/orgsettings/orgsettings.go
- **Verification:** the Task 3 integration subtest PUTs the key and asserts it is stored; the availability service unit battery reads it.
- **Committed in:** 96b356c, 239da0f

**2. [Rule 3 - Blocking] orgsvc.NewService has FIVE call sites, not the plan's three**
- **Found during:** Task 3 GREEN (compile)
- **Issue:** The plan pinned three call sites (main.go:122, handler_test_helper.go:85, main_test.go:125) but `organization_integration_test.go` and `organization_test.go` also construct the service — the constructor change broke them (compile-forced).
- **Fix:** Updated both extra call sites; the integration fixture builds the real availability service chain (D-G parity), the mock test passes nil (its surface never touches the schedule path).
- **Files modified:** internal/core/services/organization/organization_integration_test.go, organization_test.go
- **Verification:** `go build ./...` + `go test ./internal/core/services/organization/` green.
- **Committed in:** 1cee434

**3. [Rule 1 - Bug] `ErrInvalidValue` did not exist in the availability domain**
- **Found during:** Task 2 RED authoring
- **Issue:** The plan names `ErrInvalidValue` as the invalid-stored-default sentinel (twice: Task 2 behavior + threat T-14g-19). The availability domain only had `ErrInvalidRequest`; returning `orgsettings.ErrInvalidValue` would have leaked a cross-domain sentinel into the handler → 500 on corrupted defaults (the 12-05 lesson).
- **Fix:** Added `availability.ErrInvalidValue` ("invalid stored value") + its JSONNames entry; the handler maps it to 400.
- **Files modified:** internal/core/domain/availability/errors.go
- **Verification:** the invalid-default battery asserts `errors.Is(err, ErrInvalidValue)`.
- **Committed in:** 23a015d, 9147cea (gofmt alignment)

**4. [Rule 2 - Missing Critical] postgres membership scans didn't surface the 024 columns**
- **Found during:** Task 2 GREEN (production-read analysis)
- **Issue:** `GetMembership` (the ResolveSchedule membership read) and `scanMemberships` (ListByUser/ListByOrg) selected only pre-024 columns — production `ResolveSchedule` would never see `contract_type_id`/`day_hours_override`, silently resolving every employee to the fallback.
- **Fix:** Both scans now select + JSONB-unmarshal the 024 columns.
- **Files modified:** internal/adapters/secondary/postgres/organization_repo.go, organization_membership_repo.go
- **Verification:** full postgres package suite green; the http battery's schedule write + read-back exercises the real scan.
- **Committed in:** 239da0f

**5. [Rule 1 - Bug] The schedule audit vocabulary constants did not exist**
- **Found during:** Task 1 RED authoring
- **Issue:** The plan's repo batteries reference `entity_type 'contract_type'` actions created/updated/deleted and (Task 3) `organization_membership` schedule_updated — no constants existed to compile the tests against.
- **Fix:** Pinned the five constants in the availability domain (vocab-first, Phase 13 precedent) before the code landed.
- **Files modified:** internal/core/domain/availability/availability.go
- **Verification:** tests reference the constants; no string literals.
- **Committed in:** 23a015d

**6. [Rule 1 - Bug] Test names didn't match the plan's verify regex**
- **Found during:** Task 2 RED (first run reported "no tests to run")
- **Issue:** The plan's verify command runs `-run 'TestContractType|TestResolveSchedule'`; my names (`TestService_ContractTypes_*`) didn't match, so the batteries silently didn't run.
- **Fix:** Renamed to `TestContractTypes_*` / `TestResolveSchedule`.
- **Files modified:** internal/core/services/availability/contract_types_test.go
- **Verification:** the plan's exact verify command now matches and passes.
- **Committed in:** 96b356c

**7. [Rule 1 - Bug] contract_types_handler.go placement**
- **Found during:** Task 3 GREEN (file-split pass)
- **Issue:** I initially added the contract-type handler methods to availability_handler.go; the plan pins `contract_types_handler.go` as its own artifact (grep-able acceptance).
- **Fix:** Split the methods + doc block into the plan-named file; writeError stays on the handler in availability_handler.go.
- **Files modified:** internal/adapters/primary/http/contract_types_handler.go (new), availability_handler.go
- **Verification:** battery re-run green after the split.
- **Committed in:** 1cee434

---

**Total deviations:** 7 auto-fixed (4 Rule 1, 2 Rule 2, 1 Rule 3)
**Impact on plan:** All fixes were required for the plan's own MUST-HAVE truths to be expressible (the org-default key could not be served, production overrides were invisible, the named sentinel didn't exist). No scope creep — the implementations follow the plan's skeletons verbatim.

## Issues Encountered

- **MockOrgMgmtRepo did not implement the extended port** — the compile surfaced it after the port grew; added the schedule-write capture stub (also serves later service-level assertions).
- **Handler battery session/org context:** a fresh `loginUser` issues a token for the user's PRIMARY org — every re-login had to re-establish the target org via `switch-organization` (the direction_test `switchToOrg` lesson); the cross-org subtest also needed the hr re-login AFTER the other-org registration (role gate fires before same-org validation).
- **File-move hiccup:** the scripted extraction of the handler methods split one function across two files (DeleteContractType's tail stayed behind) — caught by the build, repaired with a targeted patch.
- **Same-second `created_at` ties** made the list ordering nondeterministic — the list battery asserts by id/name map, not position.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-07 (capacity read-model):** `ResolveSchedule` is live with exported level constants + `ContractTypeID` on the resolution — the capacity plan consumes the per-employee schedule + resolution level per the key_link; the repo read-model stubs (Windows/Capacity/Attachments/ActivityWorkloadEmployees) remain the clean RED surface.
- **14-08 (HTTP surface):** the contract-type routes + writeError sentinels are wired; the fixture mirrors main.go; `go test ./cmd/server/` stays green with the orgsvc constructor updated here (14-08 Task 3 only verifies call sites).
- **Phase 19 history filters:** the audit vocabulary now includes contract_type created/updated/deleted + organization_membership schedule_updated (entity_id = type id / membership id).
- Full suite green (`make test` exit 0, all packages).

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*

## Self-Check: PASSED

- All 6 planned files exist on disk (contract_type_repository.go/.test.go, availability contract_types.go/.test.go, http contract_types_handler.go/.test.go).
- All 7 plan commits verified in git history (23a015d, 824f66e, 96b356c, 239da0f, f9d95f2, 1cee434, 9147cea).
- Plan-level verification re-run: `go test ./internal/adapters/secondary/postgres/ -run TestContractType` ok; `go test ./internal/core/services/availability/ -run 'TestContractType|TestResolveSchedule'` ok; `go test ./internal/adapters/primary/http/ -run 'TestContractType|TestAvailabilityMembership'` ok; `go build ./...` ok; `go test ./cmd/server/` ok; `make test` full-suite exit 0.
- TDD gate compliance: 3 RED (`test(14-06)`) + 3 GREEN (`feat(14-06)`) commits, each RED immediately preceding its GREEN (verified via `git log --grep`).
