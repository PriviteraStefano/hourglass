---
phase: 13-direction-backend-the-plan-plane
plan: 08
subsystem: api
tags: [go, http, direction, origin-fallback, activity, cmd-server, hexagonal]

# Dependency graph
requires:
  - phase: 13-04
    provides: org_settings handler + literal /organizations/settings routes + newHandlerFixture + cmd/server wiring pattern
  - phase: 13-06
    provides: FirstDirectionRefs + full DirectionRepository interface (read-models)
  - phase: 13-07
    provides: directionsvc.Service with the pinned 7-arg constructor + warning overlay + ListPlan/Coverage read-model assembly
provides:
  - The seven direction HTTP routes (create/activate/cancel/claims/claim-cancel + plan + coverage) with the sentinel map (404/400/403/409/500)
  - The origin fallback in the activity read path (OriginType == nil → FirstDirectionRefs derivation, read-only)
  - The cmd/server + fixture wiring (direction service with shared orgsettings/routing deps; activityService constructor with the fallback seam)
affects: [13-09, phase-19-direction-ui, verify-work phase 13]

# Actuals (#2632) — pairs with the plan's `estimate` (40000 tokens) on the same scale.
actuals:
  tokens: 14172   # 56687 diff chars / 4 over the realized diff (26d9784..HEAD)
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Thin HTTP adapter: ctx claims → uuid.Parse PathValue ids → json decode → service call → pkg/api envelope (coverage_handler skeleton)"
    - "writeError sentinel switch (404/400/403/409/500) — ErrClaimOverBudget/ErrInvalidTransition surface as 409, never 500 (T-13-29)"
    - "Origin fallback: A4 predicate (OriginType == nil) → FirstDirectionRefs → response-only derivation, never written back (D-13-34)"
    - "Warnings always an array at the API boundary: nil service overlays normalize to [] (D-13-03/13-UI-SPEC)"

key-files:
  created:
    - internal/adapters/primary/http/direction_handler.go
    - internal/adapters/primary/http/direction_handler_test.go
    - internal/core/services/activity/activity_origin_fallback_test.go
  modified:
    - cmd/server/main.go
    - cmd/server/main_test.go
    - internal/adapters/primary/http/handler_test_helper.go
    - internal/core/services/activity/activity.go
    - internal/core/services/activity/activity_test.go
    - internal/core/services/activity/activity_origin_test.go

key-decisions:
  - "Origin fallback lives in the activity read path at the service layer: GetByID + List apply the OriginType == nil predicate and derive assigned_by/assigned_to from FirstDirectionRefs on the response only — read-only, origin_type stays NULL, stored refs authoritative (D-13-32..34, FND-04, Pitfall 5)"
  - "Create response shape {row, warnings} with warnings normalized to an always-array at the handler boundary (D-13-03/13-UI-SPEC); read-models carry rows (+totals) + warnings with the same array guarantee"
  - "The seven direction routes are middleware.Auth-wrapped; the direction service reuses the SHARED orgsettings + routing services — no second instances (D-G parity)"
  - "The direction repo is declared before the activities block in cmd/server (the activity service needs it as a constructor arg); the direction service itself is built after the orgsettings block (13-07 constructor pin)"

patterns-established:
  - "Warnings overlay rides every direction response (create + plan + coverage) as a JSON array — never null, never blocking (D-13-28)"
  - "TDD per behavior: failing integration battery committed first (unregistered routes → 404 RED), handler + fixture wiring land after (GREEN)"
  - "Integration battery cookie-jar discipline: registerUserInOrg/loginUser re-establish the switched org (a fresh login issues a primary-org token)"

requirements-completed: [DIR-01, DIR-02, DIR-03, DIR-05, DIR-06, FND-04]

coverage:
  - id: D1
    description: "Origin fallback in the activity read path: GetByID + List derive assigned_by/assigned_to from the first direction row when OriginType is nil; stored origin refs stay authoritative; nothing is written back"
    requirement: FND-04
    verification:
      - kind: unit
        ref: "internal/core/services/activity/activity_origin_fallback_test.go#TestActivityOriginFallback_GetByID"
        status: pass
      - kind: unit
        ref: "internal/core/services/activity/activity_origin_fallback_test.go#TestActivityOriginFallback_List"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/GET_/activities_derives_assigned_by/assigned_to_from_the_first_direction_row"
        status: pass
    human_judgment: false
  - id: D2
    description: "Direction create/activate/cancel/claims/claim-cancel routes with the pinned sentinel map (404/400/403/409/500) and the {row, warnings} create response"
    requirement: DIR-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler (permission matrix + sentinel subtests)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Supersede chain over HTTP: create-with-supersedes flips the target to superseded (history via GET /direction); superseding a claim-row target carries origin_direction_id (ADR-BE-018 §5)"
    requirement: DIR-02
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/create-with-supersedes_flips_the_target_to_superseded_(history_via_GET_/direction)"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/superseding_a_claim-row_target_carries_origin_direction_id"
        status: pass
    human_judgment: false
  - id: D4
    description: "WG claim model over HTTP: non-member claim 403, over-budget claim 409 (Σ guard under tx lock)"
    requirement: DIR-03
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/wg_claim_by_non-member_is_forbidden"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/claim_over_budget_is_409"
        status: pass
    human_judgment: false
  - id: D5
    description: "Warning overlay + plan/coverage read-models over HTTP: warnings array in the create/read responses, coverage rows + totals + warnings, read gates at the boundary (org-wide plan manager-only, coverage employee scope self-only for non-managers), period bounds required"
    requirement: DIR-05
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/read_gates:_org-wide_plan_manager-only,_employee_id=self_allowed"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/coverage_scope=employee_returns_rows_+_totals_+_warnings"
        status: pass
    human_judgment: false
  - id: D6
    description: "Direction-coverage read-model endpoint (scope params + period bounds) with the service-level scope gates"
    requirement: DIR-06
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/coverage_scope=employee_returns_rows_+_totals_+_warnings"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/coverage_scope=unknown_is_400"
        status: pass
    human_judgment: false

# Metrics
duration: 2h 23m
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 8: Direction HTTP Surface + Origin Fallback Summary

**The seven direction routes live end-to-end over HTTP (create with {row, warnings}, explicit activation, reason-required cancel/unclaim, WG claims with the Σ 409 guard, plan + coverage read-models with the warning overlay and read gates), and FND-04 lands: activities with empty origin refs derive manager-assignment refs from the first direction record on the activity read path — read-only, stored refs authoritative.**

## Performance

- **Duration:** 2h 23m
- **Started:** 2026-08-08T14:27:00Z
- **Completed:** 2026-08-08T16:50:00Z
- **Tasks:** 3
- **Files modified:** 9 (3 created, 6 modified)

## Accomplishments

- **Origin fallback (FND-04, Pitfall 5 closed):** `activity.Service` gains `directionRefs ports.DirectionRepository`; GetByID + List apply the A4 predicate (OriginType == nil), call FirstDirectionRefs, and set AssignedBy/AssignedTo on the response only — derived, never written back, origin_type stays NULL (D-13-32..34, OQ3). Proven at the service layer (unit contract tests: derived refs appear, empty refs stay empty, stored origin_type blocks the fallback with a call-counter, List enriches per-row) AND at the HTTP boundary (GET /activities carries derived assigned_by/assigned_to; stays empty without rows; stored refs authoritative).
- **Direction handler (7 routes):** `direction_handler.go` implements Create/Activate/Cancel/Claim/Unclaim/ListPlan/Coverage as thin coverage-handler-skeleton adapters — ctx claims, uuid.Parse PathValue ids, json decode, service call, pkg/api envelope. The writeError switch pins the sentinel map: ErrDirectionNotFound → 404; ErrInvalidRequest/ErrInvalidHours/ErrInvalidTarget/ErrCancelReasonRequired → 400; ErrForbidden/ErrNotWgMember → 403; ErrInvalidTransition/ErrClaimOverBudget/ErrWgRowNotActive → 409; default → 500 (T-13-29/32). Period bounds parse as 2006-01-02 (OQ5); malformed input never 500s.
- **Warnings contract:** the create response is `{row, warnings}` (D-13-03) and every direction response (create + plan + coverage) carries `warnings` as a JSON array — nil overlays normalize to `[]` at the handler boundary (never null, never blocking, D-13-28/13-UI-SPEC).
- **Wiring (Pitfall 5, one task, compile-forced):** `cmd/server/main.go` constructs `directionRepo → directionsvc.NewService(directionRepo, activityRepo, wgRepo, unitRepo, orgRepo, orgSettingsService, routingSvc)` (13-07 pin, shared orgsettings/routing — D-G parity), `http.NewDirectionHandler`, registers the seven routes with `middleware.Auth` (T-13-28), and passes directionRepo into the activityService constructor (fallback seam). The 13-04 literal `/organizations/settings` registrations are untouched (Pitfall 6). `handler_test_helper.go` mirrors the wiring so the battery exercises the real stack; `cmd/server/main_test.go` call site updated.
- **Integration battery (15 subtests):** permission matrix (self-direction 200 + warnings, cross-employee manager_planned 403, non-member claim 403, over-budget claim 409, activate-cancelled 409, unauthenticated 401), sentinel mapping (404/400/400), supersede chain (superseded target absent from the plan read; claim-row supersede carries origin_direction_id), period-bounds 400s, read gates (org-wide manager-only, employee_id=self allowed, coverage employee scope self-only), coverage rows+totals+warnings, unknown scope 400, and the fallback e2e.

## Task Commits

Each task was committed atomically (TDD test → feat per behavior):

1. **Task 1: Origin fallback in the activity read path** - `d5516ac` (test: failing origin fallback contract tests), `244e89e` (feat: fallback in GetByID/List + constructor + call sites)
2. **Task 2: cmd/server + fixture wiring** - `a6154d6` (feat: direction service/handler + 7 routes + activityService fallback seam)
3. **Task 3: Direction handler + integration battery** - `74f2a3e` (test: failing handler battery), `0fb5419` (feat: handler implementation), `a6c3e73` (fix: warnings array normalization + battery corrections)

## Files Created/Modified

- `internal/adapters/primary/http/direction_handler.go` - DirectionHandler (Create/Activate/Cancel/Claim/Unclaim/ListPlan/Coverage) + writeError sentinel map + period/uuid boundary parsing
- `internal/adapters/primary/http/direction_handler_test.go` - the 15-subtest integration battery (permission matrix, sentinels, supersede chain, read gates, coverage, fallback e2e)
- `internal/core/services/activity/activity_origin_fallback_test.go` - fallback unit contract tests (derivation, emptiness, authority, List enrichment)
- `internal/core/services/activity/activity.go` - Service.directionRefs field + NewService param + GetByID/List fallback enrichment
- `internal/core/services/activity/activity_test.go` / `activity_origin_test.go` - NewService call sites updated
- `cmd/server/main.go` - direction wiring + 7 routes + activityService constructor with directionRepo
- `cmd/server/main_test.go` - call site updated
- `internal/adapters/primary/http/handler_test_helper.go` - fixture mirror (direction repo/service/handler + 7 routes; activityService constructor)

## Decisions Made

- **Origin fallback is service-layer, response-only:** the A4 predicate (OriginType == nil) triggers FirstDirectionRefs; derivation sets response fields only — no repo write exists in the path, so T-13-30 is closed by construction (D-13-34).
- **Warnings normalized to array at the handler boundary:** nil service overlays become `[]` so the API never emits `warnings: null` — the 13-UI-SPEC array contract holds for create, plan, and coverage responses.
- **directionRepo declared before the activities block** in cmd/server (the activity service needs it as a constructor argument); the direction service itself is constructed after the orgsettings block per the 13-07 pin — a compile-forced declaration order, no behavior change.
- **Claims return the row directly** (not wrapped in {row, warnings}) — the create-only wrapper is the D-13-03 shape; read responses are the service's PlanResponse/CoverageResponse.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] cmd/server/main_test.go extra activitysvc.NewService call site**
- **Found during:** Task 2 (wiring verification)
- **Issue:** `go vet ./cmd/server/` failed — the server test file also constructs the activity service with the old 6-arg signature; the plan's file list did not include it.
- **Fix:** Passed `postgres.NewDirectionRepository(pool)` as the fallback seam argument.
- **Files modified:** cmd/server/main_test.go
- **Verification:** `go build ./... && go vet ./cmd/server/` clean; `make test` green (cmd/server package).
- **Committed in:** a6154d6 (Task 2 commit)

**2. [Rule 1 - Bug/Contract] Warnings emitted as null instead of an array**
- **Found during:** Task 3 (battery run — the create response carried `warnings: null` when the overlay was empty)
- **Issue:** The service returns a nil warnings slice when no warnings compute; the D-13-03/13-UI-SPEC contract pins an array.
- **Fix:** Normalized nil → `[]directiondomain.Warning{}` at the handler boundary for create, plan, and coverage responses.
- **Files modified:** internal/adapters/primary/http/direction_handler.go
- **Verification:** Battery subtests asserting the warnings array pass; the full http package suite green.
- **Committed in:** a6c3e73

**3. [Test-only corrections] Battery fixes found during the GREEN run**
- **Found during:** Task 3 (first GREEN runs)
- **Issue:** (a) shared cookie jar left on an employee after `registerUserInOrg`, so later owner operations 403'd — a fresh `loginUser` issues a primary-org token and loses the switched org; (b) seeded unit code 'DU' collided on `idx_units_org_code` across subtests; (c) the claim-row supersede assertion compared origin_direction_id against the working_groups table id instead of the WG direction row; (d) the claim response returns the row directly, not wrapped in {row}.
- **Fix:** Re-login + explicit switchToOrg after member re-login; randomized unit codes; assert against wgRowID; extract claim id from the direct row.
- **Files modified:** internal/adapters/primary/http/direction_handler_test.go
- **Verification:** Full battery 15/15 green; http package suite green.
- **Committed in:** a6c3e73

**4. [Execution ordering - compile dependency] Task 2/Task 3 interleaved commits**
- **Found during:** Task 2 wiring
- **Issue:** `cmd/server/main.go` + the fixture reference `http.NewDirectionHandler` (Task 3's file), so the wiring cannot compile before the handler exists; equally the battery (Task 3) cannot pass before the fixture registers the routes (Task 2).
- **Fix:** Committed in dependency order — handler (0fb5419) before wiring (a6154d6); the failing battery (74f2a3e) preceded both per TDD. Each commit preserves the plan's file ownership; the plan's own "compile-forced" note anticipated this.
- **Verification:** Every commit after the two RED test commits builds; the final state is fully green.
- **Committed in:** 74f2a3e, 0fb5419, a6154d6

**5. [Execution ordering - scoped build gate] Task 1 GREEN left the whole-repo build broken until Task 2**
- **Found during:** Task 1 close-out
- **Issue:** The activity NewService signature change (Task 1) breaks the cmd/server + fixture call sites; the plan's Task 1 acceptance criteria scope the build to the activity packages, and Task 2 explicitly owns the call-site updates ("compile-forced").
- **Fix:** Followed the plan's scoping — the repo compiles end-to-end at a6154d6 (Task 2).
- **Verification:** `go build ./...` clean at the Task 2 commit; `make test` green.
- **Committed in:** 244e89e (Task 1 GREEN, scoped), a6154d6 (repo restored)

---

**Total deviations:** 5 (2 auto-fixed issues, 2 test corrections, 1 compile-ordering note; the ordering items are plan-sanctioned)
**Impact on plan:** All fixes necessary for correctness/contract fidelity; no scope creep, no files outside the plan's declared scope (cmd/server/main_test.go is inside cmd/server, Task 2's package).

## Issues Encountered

- The shared cookie jar across subtests silently changed the acting user (registerUserInOrg switches orgs). Resolved with explicit re-login + `switchToOrg` helpers — documented in the battery as a standing pattern.
- None else — the plan's assumptions held (manager_planned default mode gate, role-gated terminal routing for unit-less employees, queued rows visible in any period).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase complete:** 8/8 plans executed; DIR-01..06 + FND-04 reachable over HTTP; full suite green (`make test` exit 0, 24 packages).
- **Ready for:** phase verification (verify-work) — the coverage block above maps every deliverable to its passing test.
- **Deferred (out of scope, pre-existing):** 13-PATTERNS.md/12-PATTERNS.md/12-VERIFICATION.md remain untracked planning files (not committed by this plan); the pre-existing dirty working-tree files (config.json, DISCUSS-CHECKPOINT.json deletion, workspace.json) were left untouched per execution constraints.

---

*Phase: 13-direction-backend-the-plan-plane*
*Completed: 2026-08-08*

## Self-Check: PASSED

- All 7 plan key-files exist on disk (handler, battery, fallback tests, activity.go, main.go, fixture, SUMMARY)
- All 6 commits verified in git history: d5516ac (test T1), 244e89e (feat T1), 74f2a3e (test T3), 0fb5419 (feat T3 handler), a6154d6 (feat T2 wiring), a6c3e73 (fix warnings/battery)
- Plan verification: `go test ./internal/adapters/primary/http/ -run TestDirectionHandler -count=1` ok; `go test ./internal/core/services/activity/ -count=1` ok; `go build ./...` + `go vet ./...` clean; `make test` exit 0 (24 packages, zero FAIL)
