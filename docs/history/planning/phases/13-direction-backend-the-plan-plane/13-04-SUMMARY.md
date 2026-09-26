---
phase: 13-direction-backend-the-plan-plane
plan: 04
subsystem: api
tags: [org-settings, jsonb, audit-log, planning-mode, go, postgres, react-query, membership]

requires:
  - phase: 13-01
    provides: org_settings table + planning_mode override column (migration 022)
  - phase: 13-03
    provides: orgsettings domain (keys/validators/errors/audit vocabulary), OrgSettingsRepository port, testdata mocks, ADR-BE-018
provides:
  - OrgSettingsRepository postgres adapter — Get/List/Upsert with the settings-updated audit row in the same tx
  - orgsettings service — known-key validated Get (code-level defaults) / Put (manager+ gate, per-key audit), ResolvePlanningMode seam
  - OrgSettingsHandler + literal GET/PUT /organizations/settings routes coexisting with the typed wildcard surface
  - OrganizationMembership.PlanningMode/ValidFrom/ValidUntil fields + GetMembership SELECT/scan extension
  - Test suites: service unit (12 cases), repo integration (5), handler battery (7)
affects: [13-07 direction service mode gate, 13-05 direction repo, verify-work]

actuals:
  tokens: 13813    # chars/4 over the realized diff (eddb165^..HEAD, 55255 chars)
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Literal-route-beside-wildcard coexistence (ServeMux most-specific-wins, Pitfall 6)"
    - "In-tx audit insert helper (insertOrgSettingsAudit mirroring insertCoverageAudit, BE-016)"
    - "Code-level defaults overlay on GET (no seed rows, ADR-BE-018 §8.3)"
    - "Service-side precedence resolution seam (override → org default → fallback)"

key-files:
  created:
    - internal/adapters/secondary/postgres/org_settings_repository.go
    - internal/core/services/orgsettings/orgsettings.go
    - internal/core/services/orgsettings/orgsettings_test.go
    - internal/adapters/secondary/postgres/org_settings_repository_test.go
    - internal/adapters/primary/http/org_settings_handler.go
    - internal/adapters/primary/http/org_settings_handler_test.go
  modified:
    - internal/core/domain/auth/membership.go
    - internal/adapters/secondary/postgres/organization_repo.go
    - cmd/server/main.go
    - internal/adapters/primary/http/handler_test_helper.go

key-decisions:
  - "ResolvePlanningMode precedence pinned: membership planning_mode override → org default planning_mode key → manager_planned fallback (D-13-19, ADR-BE-018 §8) — invalid vocabulary in either store position is ErrInvalidValue, never a silent default"
  - "Audit row payload {key, before, after} with entity_id = the ORG id is written service-side and handed to the repo for the in-tx write (D-13-22)"
  - "ValidFrom/ValidUntil/PlanningMode added as pointer fields with omitempty JSON tags — additive, NewOrganizationMembership untouched (NULL = unset)"

patterns-established:
  - "Org settings writes: manager+ gate BEFORE validation, full batch validation BEFORE any write (no partial commits)"
  - "Failed-tx rollback leaves no value row AND no audit row (T-13-12) — asserted by test"

requirements-completed: [DIR-04]

coverage:
  - id: D1
    description: "org_settings vertical end-to-end: HTTP → service validation → tx upsert + in-tx audit → read-back, manager-gated"
    requirement: DIR-04
    verification:
      - kind: e2e
        ref: "internal/adapters/primary/http/org_settings_handler_test.go#TestOrgSettingsHandler"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/org_settings_repository_test.go#TestOrgSettingsRepository_Upsert_WritesAuditInSameTx"
        status: pass
    human_judgment: false
  - id: D2
    description: "Literal GET/PUT /organizations/settings routes coexisting with typed GET/PUT /organizations/{id}/settings (Pitfall 6 closed)"
    verification:
      - kind: e2e
        ref: "internal/adapters/primary/http/org_settings_handler_test.go#TestOrgSettingsHandler_RouteCoexistence"
        status: pass
    human_judgment: false
  - id: D3
    description: "Membership PlanningMode/ValidFrom/ValidUntil fields + GetMembership scan extension (consumed by 13-07 mode gate and validity warnings)"
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/organization_repo_test.go#TestOrganizationRepository_AddMembership_GetMembership"
        status: pass
    human_judgment: false
  - id: D4
    description: "ResolvePlanningMode precedence seam for the direction service mode gate (13-07): override → org default → manager_planned fallback"
    verification:
      - kind: unit
        ref: "internal/core/services/orgsettings/orgsettings_test.go#TestOrgSettingsService_ResolvePlanningMode"
        status: pass
    human_judgment: false
  - id: D5
    description: "Permission + validation matrix: manager 200 / finance+employee 403 / unknown key 400 / invalid value 400 / bad JSON 400 / 401 unauthenticated"
    verification:
      - kind: e2e
        ref: "internal/adapters/primary/http/org_settings_handler_test.go#TestOrgSettingsHandler_PermissionMatrix"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 04: Org Settings Vertical — Tracer + Membership Extension Summary

**org_settings vertical slice: literal-route settings GET/PUT with known-key validation, manager+ gate, in-tx audit, ResolvePlanningMode seam, and the membership planning_mode/validity struct+scan extension — proven end-to-end by 3 test suites (service unit, repo integration, handler battery)**

## Performance

- **Duration:** 40 min
- **Started:** 2026-08-08T12:47:00Z (tracer + checkpoint, prior session) / continuation 2026-08-08T13:27:08Z
- **Completed:** 2026-08-08T13:27:08Z
- **Tasks:** 3 (1 tracer + 2 auto, TDD)
- **Files modified:** 10 Go files (1151 insertions)

## Accomplishments

- **Task 1 (tracer, prior session):** org_settings repo (Get/List/Upsert with the `settings-updated` audit row in the same tx), orgsettings service (known-key validation, manager+ gate, defaults overlay), handler with literal GET/PUT `/organizations/settings` routes (JWT-resolved org — no path param), cmd/server + test-fixture wiring, and the e2e test proving PUT → audit → GET round trip.
- **Task 2:** `OrganizationMembership` gained `PlanningMode *string`, `ValidFrom *time.Time`, `ValidUntil *time.Time` (additive, omitempty); `GetMembership` SELECT+scan extended with `valid_from, valid_until, planning_mode`; service exposes `ResolvePlanningMode(ctx, orgID, employeeID)` with the D-13-19 precedence (membership override → org default key → `manager_planned` fallback), invalid vocabulary → `ErrInvalidValue`.
- **Task 3:** repo integration suite (upsert create/update, in-tx audit assertions, **failed-tx rollback leaves no value AND no audit row** — T-13-12, org-scoped List, absent-key nil) + handler battery (literal/typed route coexistence — the Pitfall 6 lock, 401 unauthenticated, one/many-key PUT, value+vocabulary validation matrix, finance 403, malformed body 400).
- DIR-04 requirement delivered in full; `ResolvePlanningMode` seam ready for the 13-07 direction mode gate.

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): End-to-end org_settings vertical** — `eddb165` (test, e2e RED), `6e1e324` (feat, GREEN)
2. **Task 2: Membership struct/scan extension + ResolvePlanningMode + unit tests** — `9e7502f` (test, RED), `a831549` (feat, GREEN)
3. **Task 3: Repo integration tests + handler battery** — `c575836` (test)

**Plan metadata:** (committed after this SUMMARY)

## Files Created/Modified

- `internal/adapters/secondary/postgres/org_settings_repository.go` — OrgSettingsRepository: Get (absent → nil,nil), List, Upsert (ON CONFLICT value-replace) + `insertOrgSettingsAudit` in-tx helper (BE-016 shape)
- `internal/core/services/orgsettings/orgsettings.go` — Get (defaults overlay: `planning_daily_hours` → 8.0), Put (manager+ gate first, full-batch validation, per-key {key,before,after} audit), `ResolvePlanningMode`
- `internal/adapters/primary/http/org_settings_handler.go` — Get/Put over the JWT-resolved org, sentinel→status writeError map
- `internal/core/domain/auth/membership.go` — PlanningMode/ValidFrom/ValidUntil pointer fields
- `internal/adapters/secondary/postgres/organization_repo.go` — GetMembership SELECT + scan extension
- `cmd/server/main.go` — literal `GET/PUT /organizations/settings` beside the typed wildcard registrations (not removed)
- `internal/adapters/primary/http/handler_test_helper.go` — orgsettings wiring in `newHandlerFixture`
- `internal/core/services/orgsettings/orgsettings_test.go` — 12 unit cases (Put gate matrix, audit shape, defaults, ResolvePlanningMode precedence)
- `internal/adapters/secondary/postgres/org_settings_repository_test.go` — 5 integration cases incl. failed-tx rollback
- `internal/adapters/primary/http/org_settings_handler_test.go` — e2e + 6 battery tests

## Decisions Made

- **ResolvePlanningMode precedence pinned** (D-13-19): membership override → org default key → `manager_planned` fallback. An invalid mode string in EITHER position is `ErrInvalidValue` — the JSONB store is unvalidated (T-13-11), so the resolution surfaces corruption rather than silently defaulting.
- **Audit row shape**: `entity_type='org_settings'`, `entity_id = org id`, `action='settings-updated'`, payload `{key, before, after}` — built service-side, written by the repo in the upsert tx (D-13-22).
- **Membership fields additive**: pointer fields + omitempty; `NewOrganizationMembership` untouched; `AddMembership` INSERT unchanged (columns nullable).
- **No seed rows**: GET overlays code-level defaults for absent keys (D-13-24, ADR-BE-018 §8.3).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- None. All three suites were green on first full run; the only hiccup was a declared-but-unused variable in the new repo test (fixed before commit) and a malformed `--no-verify=false` flag invocation on the RED commit (re-run without the flag — hooks ran normally on all commits).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Org settings vertical proven end-to-end; `ResolvePlanningMode` seam available for the 13-07 direction service mode gate.
- Membership validity fields (`valid_from`/`valid_until`) scanned and on the struct — the 13-07 validity warnings read them.
- Direction repository (`direction_repository.go`) intentionally untouched — 13-05 owns it.

## Self-Check

- `go test ./internal/core/services/orgsettings/ ./internal/adapters/secondary/postgres/ ./internal/adapters/primary/http/ -run 'TestOrgSettings|TestOrganizationRepository' -count=1` — PASS (exit 0)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `make test` (full suite) — PASS (23 packages ok, 0 failures)
- Task 1 commits verified present: `eddb165`, `6e1e324`
- Acceptance criteria for Tasks 2 and 3 verified individually during execution (see coverage block)

---
*Phase: 13-direction-backend-the-plan-plane*
*Completed: 2026-08-08*
