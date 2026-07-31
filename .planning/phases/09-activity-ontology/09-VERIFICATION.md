---
phase: 09-activity-ontology
verified: 2026-07-31T17:40:31Z
status: gaps_found
score: 12/15 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Subproject-derived activities carry kind = 'phase' (SPEC acceptance #6)"
    status: failed
    reason: "Migration 011 maps subprojects to kind='task' (not the SPEC-locked 'phase'). 09-01-PLAN.md task 1 said 'task (review seed to confirm)', but the SPEC interview explicitly locked 'phase' after seed review. Executor followed the plan wording; summary documents kinds 6/6/1 (engagement/task/internal) without flagging the SPEC conflict."
    artifacts:
      - path: "migrations/011_activity_ontology.up.sql"
        issue: "Line 115: subproject INSERT hardcodes kind='task'; SPEC acceptance #6 requires 'phase'"
    missing:
      - "Either flip the subproject migration to kind='phase' (and the test assertion), or accept a formal override (kind is a free label per D-2 — no depth constraint)"
  - truth: "go test ./... green (SPEC acceptance #12, phase boundary c)"
    status: failed
    reason: "18/19 packages pass; internal/core/services/working_group is red. TestWorkingGroupIntegration seeds the projects/subprojects tables dropped by migration 011 (SQLSTATE 42P01). Regression caused by this phase's migration, deferred to the Phase 10 working_group renovation per deferred-items.md — not fixed in this phase."
    artifacts:
      - path: "internal/core/services/working_group/working_group_integration_test.go"
        issue: "Lines 48/55: seedWGData INSERTs into dropped projects/subprojects tables"
    missing:
      - "Re-seed the test with activity_kinds + activities (fix suggested in deferred-items.md, small and self-contained)"
  - truth: "Cycle prevention on activities.parent_id (SPEC in-scope item)"
    status: failed
    reason: "SPEC 'In scope' lists 'Cycle prevention on activities.parent_id (path check on insert/update)' — no implementation found in the activity service or repository. No plan task claimed it; the in-scope item was dropped without documentation."
    artifacts:
      - path: "internal/adapters/secondary/postgres/activity_repository.go"
        issue: "Create/Update accept any parent_id (parent same-org checked in service); no ancestor-path check prevents cycles"
      - path: "internal/core/services/activity/activity.go"
        issue: "Create validates parent exists + same-org only; Update does not even validate parent same-org"
    missing:
      - "Path check on insert/update (walk GetAncestry of the new parent; reject if it contains the activity's own id)"
deferred:
  - truth: "All existing Go tests pass against the new schema (CONTEXT phase boundary c; SPEC acceptance #12)"
    addressed_in: "Phase 10"
    evidence: "deferred-items.md item 2 assigns the working_group test re-seed to 'the working_group renovation / phase 10 field rename'; Phase 10 goal includes 'Working Groups surface'. The failing package is the working_group service, whose SubprojectID field rename is itself deferred to Phase 10 (code comment in working_group.go)."
human_verification:
  - test: "Run the server against a testcontainers/live PostgreSQL and smoke-test the full activity flow end-to-end: POST /activities (create parent + child), POST /time-entries with activity_id, submit, verify routing to WG manager/delegate, approve."
    expected: "Create/list/children/detail endpoints return ancestry + commercial_context + billable; entry submit on a commercial activity without an anchored WG returns 409 with the ErrActivityNotLoggable message; WG manager approve transitions to pending_finance."
    why_human: "Requires a running server, DB, and authenticated session — handler integration tests cover this programmatically, but the live-server smoke the executor claimed in 09-05 is not reproducible from static inspection."
---

# Phase 9: Activity Ontology — Verification Report

**Phase Goal:** Replace `projects`+`subprojects` with the recursive `activities` entity, rewrite all FKs and approval routing onto the new ontology, land the additive staffing schema (ADR-P-008), and leave the backend API fully activity-shaped. Backend + database only — frontend rename and new surfaces are Phase 10.

**Verified:** 2026-07-31T17:40:31Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

The big-bang ontology rewrite is **substantially achieved and verifiable in code**: the recursive `activities` schema replaces the two-level model, all six FK rewrites land in one atomic migration with zero orphaned seed rows, the domain/port/adapter collapse to a single `Activity` entity + `activity_repository` is complete (old project/subproject files deleted), approval routing for both entry types resolves through the activity → WG → manager/delegate chain with unit-manager fallback and `ErrActivityNotLoggable` enforcement, the staffing schema is live at the DB level, and the HTTP surface is fully activity-shaped with old routes removed. `go build ./...`, `go vet ./...`, and 18/19 test packages are green.

Three gaps remain: (1) a locked SPEC decision (subproject-derived activities `kind='phase'`) was implemented as `'task'` without flagging the deviation; (2) `go test ./...` is not fully green — the working_group integration test (broken by this phase's own migration 011) was deferred rather than fixed; (3) the SPEC in-scope cycle-prevention item on `activities.parent_id` was never implemented.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Recursive `activities` schema replaces `projects`+`subprojects`; `activity_kinds` org-extensible catalog (P-007 D-1/D-2) | ✓ VERIFIED | `011_activity_ontology.up.sql` creates activities (self-ref parent_id ON DELETE RESTRICT, nullable contract_id, nullable billable, kind FK to catalog); drops projects/subprojects/project_adoptions/project_managers; seeds 4 kinds for MVP org |
| 2 | Commercial context optional + inherited upward via CTE; `billable` nullable (D-3/D-7) | ✓ VERIFIED | `ResolveCommercialContext`/`ResolveBillability` recursive CTEs in postgres/activity_repository.go; tested against 3-level tree (13 tests in activity_repository_test.go) |
| 3 | Entries link through one `activity_id NOT NULL`; wg_id/project_id/subproject_id/customer_id dropped (D-4, R-5) | ✓ VERIFIED | Migration rewrites both entry tables; domain time_entry.go/expense.go have ActivityID+UnitID only; entry repos use activity_id SQL only; zero-orphan asserted by migration test |
| 4 | `working_groups.activity_id` replaces `subproject_id`; `enforce_unit_tuple` dropped from schema (D-5, R-5) | ✓ VERIFIED | Migration renames column, drops column; working_group_repository.go queries activity_id; no SQL residue |
| 5 | Approval routing: activity → anchored WG → manager/delegate; unit-manager fallback; D-11 skip incl. delegates; `ErrActivityNotLoggable` (R-1…R-3) | ✓ VERIFIED | resolveManagerStage/resolveUnitManager in both services; skipToFinance; sentinel in activity domain; full test matrix in both services, `TestService_Submit_Routing` PASS (verified by running) |
| 6 | Repository collapse: one `activity_repository` port + adapter (R-6) | ✓ VERIFIED | ports/activity_repository.go single interface; postgres/activity_repository.go full adapter; old domain/port/adapter files deleted; `var _ ports.ActivityRepository` assertion |
| 7 | Staffing schema (P-008): `availability_windows`, membership validity dates, `hr` role | ✓ VERIFIED | `012_staffing_schema.up.sql` creates table with kind/status CHECKs + index, 3 nullable DATE columns, role CHECK extended; cycle test TestMigration012 in package (package green in full suite) |
| 8 | Migration up/down apply cleanly; up→down→up cycle clean (testcontainers) | ✓ VERIFIED | `TestMigration011_ActivityOntology_UpDownUpCycle` PASS (verified by running); down migration documented lossy per ADR-BE-004 |
| 9 | Seed data migrated zero-orphan (every entry has a valid activity_id) | ✓ VERIFIED | Migration test asserts zero orphans on both tables; kind distribution 6/6/1 (engagement/task/internal fallback); 13 activities |
| 10 | Subproject-derived activities carry `kind = 'phase'` (SPEC acceptance #6) | ✗ FAILED | Migration line 115 hardcodes `'task'`; SPEC interview locked `'phase'` after seed review (SPEC ambiguity report + interview log) |
| 11 | `go build ./...` succeeds; no `project_repository`/`subproject_repository` symbols remain | ✓ VERIFIED | `go build ./...` exit 0; `go vet ./...` clean; grep confirms no project/subproject repo symbols in live code |
| 12 | `/api/activities` endpoint set live; `/api/projects`/`/api/subprojects` removed | ✓ VERIFIED | cmd/server/main.go registers GET/POST /activities, GET/PUT/DELETE /activities/{id}, GET /activities/{id}/children, GET /activity-kinds (all behind middleware.Auth); no /projects or /subprojects routes |
| 13 | Routing/visibility tests pass incl. `ErrActivityNotLoggable` → 409 | ✓ VERIFIED | Both entry handlers map sentinel → 409 via errors.Is; R-4 unit-subtree gating is repo-side (postgres package green); routing matrix PASS |
| 14 | `go test ./...` green (SPEC acceptance #12) | ✗ FAILED | Full run: 18/19 packages pass; `internal/core/services/working_group` FAIL — TestWorkingGroupIntegration seeds dropped tables (SQLSTATE 42P01). Deferred to Phase 10 WG renovation |
| 15 | Frontend quarantined: no `web/src/` changes; known-broken build accepted | ✓ VERIFIED | git log in phase commit range shows zero web/src modifications; SPEC excludes frontend build from pass criteria ("known-broken until Phase 10") |

**Score:** 12/15 truths verified

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | `go test ./...` fully green — working_group integration test re-seed | Phase 10 | deferred-items.md item 2: "re-seed with activity_kinds + activities… belongs with the working_group renovation / phase 10 field rename"; Phase 10 goal includes "Working Groups surface"; working_group.go field comments defer SubprojectID rename to phase 10 |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | --------- | ------ | ------- |
| `migrations/011_activity_ontology.up.sql` | activities/activity_kinds/activity_adoptions/activity_managers; FK rewrites; seed migration; drop old tables | ✓ VERIFIED | 230 lines; all six FK rewrites (entries, WGs, cutoff periods, budget_caps) + data migration + `_expense_fallback`; indexes on parent_id + working_groups(activity_id) |
| `migrations/011_activity_ontology.down.sql` | Reverse schema + data, documented lossy | ✓ VERIFIED | Restores two-level model 1:1; lossiness header; cycle test PASS |
| `migrations/012_staffing_schema.up.sql` | availability_windows + 3 columns + hr role | ✓ VERIFIED | Matches ADR-P-008 sketch exactly; drop+recreate role CHECK |
| `migrations/012_staffing_schema.down.sql` | Exact reversal | ✓ VERIFIED | Downgrades hr rows before restoring CHECK (documented accommodation) |
| `internal/core/domain/activity/activity.go` | Full P-007 field set, ActivityKind string type, DTOs, sentinels | ✓ VERIFIED | 123 lines; includes ErrActivityNotLoggable (shared by both entry services) |
| `internal/core/ports/activity_repository.go` | Single port: CRUD, ListChildren, GetAncestry, ResolveCommercialContext, ResolveBillability, KindExists, ListKinds, guards | ✓ VERIFIED | 49 lines; complete interface |
| `internal/adapters/secondary/postgres/activity_repository.go` | Full adapter with recursive CTEs | ✓ VERIFIED | 658 lines; ancestry/commercial/billability CTEs + KindExists + ListKinds + subtree delete guards |
| `internal/core/services/activity/activity.go` | ActivityService CRUD + D-2/D-3 validation + sentinel-guarded Delete | ✓ VERIFIED | 183 lines; 19 sub-tests in activity_test.go |
| `internal/adapters/primary/http/activity_handler.go` | 7 endpoints; detail composes ancestry + commercial context + billable | ✓ VERIFIED | 355 lines; sentinel→HTTP mapping; 8 boundary unit tests + integration test in handler_integration_test.go |
| `cmd/server/main.go` | Router rewired to activity endpoint set | ✓ VERIFIED | /activities + /activity-kinds registered; no project/subproject routes |
| Deleted files | project/subproject domain, ports, adapters, services, handlers | ✓ VERIFIED | domain/project/, ports/{project,subproject}_repository.go, postgres/{project,subproject}_repository.go, services/project/, http/project.go all gone |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Service Submit | Manager stage | `resolveManagerStage` (activity → WG → manager+delegates) | ✓ WIRED | Both entry services; tests PASS |
| Service Submit | Unit-manager fallback | `resolveUnitManager` upward unit-tree walk | ✓ WIRED | Personal activities (no contract, no WG) route to unit manager |
| Service Submit | ErrActivityNotLoggable | sentinel on commercial-without-WG | ✓ WIRED | Both services return it; both handlers map to 409 |
| Handler | ActivityService + repo | `ActivityHandler` holds service + repo port | ✓ WIRED | Detail endpoint composes GetAncestry/ResolveCommercialContext/ResolveBillability |
| Router | Handlers | `mux.HandleFunc` in main.go | ✓ WIRED | All 7 activity endpoints behind middleware.Auth |
| Entry repos | ListPending chain | activity → WG join | ✓ WIRED | Both repos; postgres package green |
| Entry repos | Subtree list filter | `WITH RECURSIVE activity_subtree` | ✓ WIRED | Both repos; matches SPEC "repository resolves the subtree" |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| ActivityHandler.Get | ancestry/commercial_context/billable | repo recursive CTEs | Yes — real DB queries | ✓ FLOWING |
| TimeEntry repo Create | activity_name/activity_kind | scalar-subquery RETURNING join | Yes — joined values in one round trip | ✓ FLOWING |
| Entry ListPending | manager approver set | WG row ManagerID+DelegateIDs | Yes — real resolution | ✓ FLOWING |
| Activity Create | kind catalog | `KindExists` EXISTS query | Yes — real catalog check | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Migration up→down→up cycle | `go test ./internal/adapters/secondary/postgres/ -run TestMigration011_ActivityOntology -v` | PASS (2.29s) | ✓ PASS |
| Routing rewrite (R-1/R-2/R-3) | `go test ./internal/core/services/time_entry/ -run TestService_Submit_Routing -v` | PASS — all 9 subtests incl. D-11 skip manager/delegate/member | ✓ PASS |
| Full backend suite | `go test ./... -count=1` | 18/19 packages PASS; working_group FAIL (pre-existing deferred) | ✗ FAIL (1 pkg) |
| Build + vet | `go build ./...`; `go vet ./...` | Both exit 0 | ✓ PASS |
| Old-route removal | `grep '/projects\|/subprojects' cmd/server/main.go` | No matches | ✓ PASS |
| Legacy-symbol removal | `grep project_repository/subproject_repository internal/ cmd/` | No matches in live code | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — phase plans/summaries declare no probe scripts (`scripts/*/tests/probe-*.sh`); verification is via Go testcontainers suite + live migrate CLI (both executed by the phase and reproduced here for key tests).

### Requirements Coverage

No plan requirement IDs (P-007-D*, P-008-D*, BE-014-R*, P-011-D*) are registered in `.planning/REQUIREMENTS.md` (0 matches — it tracks milestone-level IDs like TEST-*/AUTH-*). Per 09-CONTEXT, the ADRs in `hourglass-vault/decisions/` are the canonical spec. Every frontmatter ID is accounted for against its ADR and code evidence:

| Requirement | Source ADR | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| P-007-D1 | ADR-P-007 | Table/domain word is `activity` | ✓ SATISFIED | activities table + domain type |
| P-007-D2 | ADR-P-007 | Recursive parent_id; kind free catalog label | ✓ SATISFIED | schema + ActivityKind string type (no enum) |
| P-007-D3 | ADR-P-007 | contract_id nullable; commercial derived via CTE | ✓ SATISFIED | ResolveCommercialContext CTE |
| P-007-D4 | ADR-P-007 | Entries reference one activity_id NOT NULL | ✓ SATISFIED | migration + domain + repos |
| P-007-D5 | ADR-P-007 | WG anchors at any depth via activity_id | ✓ SATISFIED | migration + WG repo |
| P-007-D6 | ADR-P-007 | Big-bang migration + data migration | ✓ SATISFIED | 011 up/down + cycle test |
| P-007-D7 | ADR-P-007 | billable nullable = inherit | ✓ SATISFIED | schema + ResolveBillability |
| P-007-D8 | ADR-P-007 | Personal activities first-class; unit-manager fallback | ✓ SATISFIED | R-2 fallback in both services |
| BE-014-R1 | ADR-BE-014 | Routing from activity_id chain | ✓ SATISFIED | resolveManagerStage both services |
| BE-014-R2 | ADR-BE-014 | Precedence; ErrActivityNotLoggable | ✓ SATISFIED | sentinel + enforcement + 409 |
| BE-014-R3 | ADR-BE-014 | D-11 skip incl. delegates | ✓ SATISFIED | skipToFinance tests PASS |
| BE-014-R4 | ADR-BE-014 | Unit-subtree visibility | ✓ SATISFIED | repo-side gating preserved |
| BE-014-R5 | ADR-BE-014 | Schema consequences (FK drops, WG, cutoff, IsPeriodLocked key) | ✓ SATISFIED | migration + IsPeriodLocked(org, activity, date) |
| BE-014-R6 | ADR-BE-014 | Repo collapse + ListPending via activity→WG | ✓ SATISFIED | single port/adapter; both repos |
| P-008-D1/D1a/D1b | ADR-P-008 | availability_windows table + CHECKs | ✓ SATISFIED | 012 migration + cycle test |
| P-008-D2 | ADR-P-008 | Membership validity dates | ✓ SATISFIED | 3 nullable DATE columns |
| P-008-D4 | ADR-P-008 | `hr` role accepted by DB | ✓ SATISFIED (DB) | role CHECK extended; Go model lacks RoleHR (see Anti-Patterns) |
| P-011-D6 | ADR-P-011 | Backend activity-shaped; frontend rename deferred | ✓ SATISFIED | /activities endpoint set live; frontend untouched |

**Orphaned requirements:** none — all frontmatter IDs mapped above. No REQUIREMENTS.md IDs map to this phase (phase carries ADR decision IDs only).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `internal/core/domain/working_group/working_group.go` | 22/36/45 | Legacy `EnforceUnitTuple`/`SubprojectID` fields survive the schema drop | ⚠️ WARNING | SPEC acceptance #4 says enforce_unit_tuple absent from "schema and Go domain type" — schema clean, Go domain + HTTP DTO still carry it (documented deferral to Phase 10; HTTP handler still accepts the field in requests) |
| `internal/models/models.go` | 130-370 | Legacy `ProjectID`-bearing types (Project, TimeEntry, Expense, ExpenseItem) | ℹ️ INFO | Dead types — referenced only by testdata factories (which now set ActivityID); no live path |
| `internal/adapters/primary/http/export.go` | 301-303 | `queryProjectID` dead helper (never called) | ℹ️ INFO | Cosmetic residue; export repo rewritten to commercial-chain CTE |
| `internal/models/surreal_models.go` | whole file | Unused legacy Surreal-era models with ProjectID | ℹ️ INFO | No importers found in internal/ or cmd/ |
| `internal/models/models.go` | 11-25 | No `RoleHR` constant; `IsValid()` rejects 'hr' | ⚠️ WARNING | DB role CHECK accepts 'hr' but Go org service (`req.Role.IsValid()`) would reject it — role unusable via API until Phase 10; 09-02 summary flagged it for 09-03, never landed |
| `migrations/011_activity_ontology.up.sql` | 115 | Subproject kind hardcoded `'task'` vs SPEC-locked `'phase'` | ⚠️ WARNING | Locked SPEC decision (interview log) implemented differently; unacknowledged in summary |
| SPEC in-scope: cycle prevention | — | `activities.parent_id` path check absent | ⚠️ WARNING | Create/Update accept any parent; recursive CTEs could loop on a cycle |

**Debt markers:** no TBD/FIXME/XXX in any phase-modified file. Frontend build (`bun run build`) fails with TS errors — this is the *expected* quarantined state per SPEC ("known-broken until Phase 10", excluded from pass criteria); no `web/src/` files were modified by this phase.

### Human Verification Required

### 1. Live end-to-end activity flow smoke test

**Test:** Run the server against a testcontainers/live PostgreSQL and exercise the full flow: create parent + child activity, list with filters, fetch detail (ancestry + commercial_context + billable), log a time entry with `activity_id`, submit, approve as WG manager.
**Expected:** All endpoints respond correctly; submit on a commercial activity without an anchored WG returns 409 with "this activity requires a working group"; D-11 skip routes owner-is-approver entries straight to `pending_finance`.
**Why human:** Requires a running server + DB + authenticated session; handler integration tests cover it programmatically, but the executor's claimed live curl smoke (09-05) is not reproducible from static inspection.

### Gaps Summary

The ontology rewrite itself — schema, data migration, domain/ports/adapters collapse, routing rewrite, staffing schema, HTTP surface — is **done and verified in code** (12/15 truths, build/vet green, 18/19 test packages green). Three gaps block a clean PASS:

1. **`kind='phase'` deviation (SPEC acceptance #6):** subproject-derived activities carry `'task'`. The plan text said 'task', but the SPEC interview explicitly locked `'phase'` after reviewing the seed data — the summary never flags the conflict. **Suggested:** flip the migration + test assertion to 'phase', or accept an override (D-2 makes kind a free label, so this is data labeling, not behavior).
2. **`go test ./...` not green:** the working_group integration test (seeding dropped tables) was broken by this phase's migration 011 and deferred to Phase 10's WG renovation instead of fixed. **Suggested:** apply the re-seed fix from deferred-items.md (small, self-contained) or accept the documented deferral.
3. **Cycle prevention on `activities.parent_id` (SPEC in-scope):** never implemented, no plan task claimed it, no documentation of the drop. **Suggested:** add an ancestor-path check in Create/Update (walk the new parent's ancestry; reject if it contains the activity's own id).

The `hr` role (P-008-D4) is DB-live but Go-unusable until `RoleHR` lands in `models.go` — the 09-02 summary flagged it for 09-03 and it was dropped; Phase 10 surfaces will need it.

---

_Verified: 2026-07-31T17:40:31Z_
_Verifier: the agent (gsd-verifier)_
