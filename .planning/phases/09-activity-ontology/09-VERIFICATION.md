---
phase: 09-activity-ontology
verified: 2026-07-31T20:45:00Z
status: human_needed
score: 16/16 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 12/15
  gaps_closed:
    - "Subproject-derived activities carry kind = 'phase' (SPEC acceptance #6) — forward migration 013 + interleaved cycle-test assertions (09-06)"
    - "go test ./... green (SPEC acceptance #12) — working_group integration test re-seeded onto activity_kinds + activities; 19/19 packages (09-07)"
    - "Cycle prevention on activities.parent_id (SPEC in-scope item) — ErrActivityCycle sentinel + shared validateParent on Create/Update + HTTP 400 (09-08)"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Run the server against a testcontainers/live PostgreSQL and smoke-test the full activity flow end-to-end: POST /activities (create parent + child), attempt a cycle-creating PUT /api/activities/{id} reparent (expect HTTP 400 'activity parent would create a cycle'), POST /time-entries with activity_id, submit, verify routing to WG manager/delegate, approve."
    expected: "Create/list/children/detail endpoints return ancestry + commercial_context + billable; a reparent into the activity's own subtree returns 400 (ErrActivityCycle), not 500; entry submit on a commercial activity without an anchored WG returns 409 with the ErrActivityNotLoggable message; WG manager approve transitions to pending_finance."
    why_human: "Requires a running server, DB, and authenticated session — handler integration tests cover this flow programmatically (http package integration tests green in the full suite), but the live-server smoke the executor claimed in 09-05 is not reproducible from static inspection, and the new ErrActivityCycle 400 path added by 09-08 is only unit-tested at the service layer."
---

# Phase 9: Activity Ontology — Verification Report (RE-VERIFICATION)

**Phase Goal:** Replace `projects`+`subprojects` with the recursive `activities` entity, rewrite all FKs and approval routing onto the new ontology, land the additive staffing schema (ADR-P-008), and leave the backend API fully activity-shaped. Backend + database only — frontend rename and new surfaces are Phase 10.

**Verified:** 2026-07-31T20:45:00Z
**Status:** human_needed (all 16/16 truths verified; 1 live-server smoke item pending human)
**Re-verification:** Yes — after gap closure (09-06/07/08 executed)

## Goal Achievement

The big-bang ontology rewrite is **fully achieved and verifiable in code**. All three gaps from the initial verification (12/15) are closed by the wave-3 gap-fix plans, each confirmed by re-running the tests:

1. **Gap 1 — `kind='phase'` (SPEC acceptance #6):** forward migration `013_activity_kind_phase_fix.up.sql` relabels subproject-derived rows (`kind='task' AND parent_id IS NOT NULL` → `'phase'`); 011/012/000 remain byte-identical (ADR-BE-004 append-only preserved). The 011 migration cycle test now interleaves 013 and asserts `phase 6 / task 0` in both up passes, with down013's exact reversal proven (`task 6 / phase 0`) before down011. **Re-ran `TestMigration011_ActivityOntology_UpDownUpCycle` → PASS (4.11s).**
2. **Gap 2 — `go test ./...` green:** `seedWGData` in the working_group integration test was re-seeded onto `activity_kinds` + `activities` (zero references to the migration-011-dropped tables). **Re-ran `TestWorkingGroupIntegration` → 4/4 subtests PASS; full `go test ./... -count=1` → 19/19 packages green** (confirmed 19 test-bearing packages via `go list`).
3. **Gap 3 — cycle prevention on `activities.parent_id` (SPEC in-scope):** `ErrActivityCycle` sentinel (ADR-BE-001) + shared `validateParent` service helper (parent exists → `ErrActivityNotFound`, same-org → `ErrInvalidRequest`, `GetAncestry` walk rejects when the chain contains the activity's own id → `ErrActivityCycle`), enforced on both Create (uuid.Nil own-id) and Update (real id — also closing the latent Update same-org gap). Handler maps `ErrActivityCycle` → HTTP 400. **Re-ran `TestService_UpdateParentValidation` (6 subtests) + `TestService_Create` (incl. path-check-on-insert subtest) → all PASS.**

`go build ./...` and `go vet ./...` both clean. No debt markers (TBD/FIXME/XXX) in any phase-modified file.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Recursive `activities` schema replaces `projects`+`subprojects`; `activity_kinds` org-extensible catalog (P-007 D-1/D-2) | ✓ VERIFIED | `011_activity_ontology.up.sql` creates activities (self-ref parent_id ON DELETE RESTRICT, nullable contract_id, nullable billable, kind FK to catalog); drops projects/subprojects/project_adoptions/project_managers; seeds 4 kinds for MVP org. Immutable — byte-identical to pre-gap-fix state |
| 2 | Commercial context optional + inherited upward via CTE; `billable` nullable (D-3/D-7) | ✓ VERIFIED | `ResolveCommercialContext`/`ResolveBillability` recursive CTEs in postgres/activity_repository.go (lines 226/261); tested against 3-level tree |
| 3 | Entries link through one `activity_id NOT NULL`; wg_id/project_id/subproject_id/customer_id dropped (D-4, R-5) | ✓ VERIFIED | Migration rewrites both entry tables (drops at lines 157-159/197-198); domain time_entry.go/expense.go have ActivityID+UnitID only; zero-orphan asserted by migration test |
| 4 | `working_groups.activity_id` replaces `subproject_id`; `enforce_unit_tuple` dropped from schema (D-5, R-5) | ✓ VERIFIED | Migration renames column (line 137), drops column (line 138); working_group_repository.go queries activity_id; no SQL residue |
| 5 | Approval routing: activity → anchored WG → manager/delegate; unit-manager fallback; D-11 skip incl. delegates; `ErrActivityNotLoggable` (R-1…R-3) | ✓ VERIFIED | resolveManagerStage/resolveUnitManager in both services; skipToFinance; sentinel in activity domain; routing matrix green in full suite (time_entry + expense service packages ok) |
| 6 | Repository collapse: one `activity_repository` port + adapter (R-6) | ✓ VERIFIED | ports/activity_repository.go single interface; postgres/activity_repository.go full adapter; old files deleted; `var _ ports.ActivityRepository` assertion |
| 7 | Staffing schema (P-008): `availability_windows`, membership validity dates, `hr` role | ✓ VERIFIED | `012_staffing_schema.up.sql` (availability_windows + CHECKs + index, 3 nullable DATE columns, role CHECK extended); `TestMigration012_StaffingSchema_UpDownUpCycle` PASS (re-ran, 0.32s) |
| 8 | Migration up/down apply cleanly; up→down→up cycle clean (testcontainers) | ✓ VERIFIED | `TestMigration011_ActivityOntology_UpDownUpCycle` PASS with 013 interleaved (re-ran, 4.11s); down migration documented lossy per ADR-BE-004 |
| 9 | Seed data migrated zero-orphan (every entry has a valid activity_id) | ✓ VERIFIED | Migration test asserts zero orphans on both tables; 13 activities total |
| 10 | Subproject-derived activities carry `kind = 'phase'` (SPEC acceptance #6) — **GAP 1 CLOSED** | ✓ VERIFIED | `013_activity_kind_phase_fix.up.sql` UPDATEs `kind='task' AND parent_id IS NOT NULL` → `'phase'`; cycle test asserts engagement 6 / phase 6 / task 0 / internal 1 in both up passes; down013 reversal proven (task 6 / phase 0). Re-ran: PASS |
| 11 | `go build ./...` succeeds; no `project_repository`/`subproject_repository` symbols remain | ✓ VERIFIED | `go build ./...` exit 0; `go vet ./...` clean (re-ran); grep confirms zero legacy repo symbols in internal/ cmd/ |
| 12 | `/api/activities` endpoint set live; `/api/projects`/`/api/subprojects` removed | ✓ VERIFIED | cmd/server/main.go registers 7 activity routes (lines 191-197, all behind middleware.Auth); no /projects or /subprojects routes |
| 13 | Routing/visibility tests pass incl. `ErrActivityNotLoggable` → 409 | ✓ VERIFIED | Both entry handlers map sentinel → 409 via errors.Is (time_entry.go:330, expense.go:342); R-4 unit-subtree gating repo-side (postgres package green) |
| 14 | `go test ./...` green (SPEC acceptance #12) — **GAP 2 CLOSED** | ✓ VERIFIED | Full `go test ./... -count=1` re-run: **19/19 packages green** — no FAIL lines; `internal/core/services/working_group` PASS (TestWorkingGroupIntegration 4/4 subtests, re-seeded onto activity_kinds + activities); cmd/migrate confirmed ok |
| 15 | Frontend quarantined: no `web/src/` changes; known-broken build accepted | ✓ VERIFIED | git log across the full phase range (incl. gap-fix commits) shows zero web/src commits; SPEC excludes frontend build from pass criteria ("known-broken until Phase 10") |
| 16 | Cycle prevention on `activities.parent_id` (SPEC in-scope item) — **GAP 3 CLOSED** | ✓ VERIFIED | `ErrActivityCycle` sentinel (domain var block, line 31); `validateParent` (service lines 124-145): exists + same-org + GetAncestry walk; Create (uuid.Nil) and Update (real id) both enforce; handler maps to HTTP 400 (activity_handler.go:291-292). `TestService_UpdateParentValidation` 6 subtests + create-with-parent subtest PASS (re-ran) |

**Score:** 16/16 truths verified

### Deferred Items

None — the single deferred item from the initial verification (working_group test re-seed) was closed by 09-07. Remaining out-of-scope notes (deferred-items.md item 1: `000_full_schema.down.sql` missing `organization_settings` from its drop list — pre-existing full-cascade `-down` CLI issue unrelated to this phase) are informational, not phase truths.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | --------- | ------ | ------- |
| `migrations/011_activity_ontology.up.sql` | activities/activity_kinds/activity_adoptions/activity_managers; FK rewrites; seed migration; drop old tables | ✓ VERIFIED | Unchanged (byte-identical to HEAD) — 230 lines; six FK rewrites + data migration + `_expense_fallback` |
| `migrations/011_activity_ontology.down.sql` | Reverse schema + data, documented lossy | ✓ VERIFIED | Restores two-level model 1:1; cycle test PASS |
| `migrations/012_staffing_schema.up.sql` / `.down.sql` | availability_windows + 3 columns + hr role / exact reversal | ✓ VERIFIED | Matches ADR-P-008 sketch; drop+recreate role CHECK; cycle test PASS (re-ran) |
| `migrations/013_activity_kind_phase_fix.up.sql` | Forward label fix kind='task' → 'phase' for subproject-derived rows | ✓ VERIFIED | One-statement UPDATE with exact predicate; header documents ADR-BE-004 immutability rationale + expected distribution |
| `migrations/013_activity_kind_phase_fix.down.sql` | Reverse label fix (phase → task), same row set | ✓ VERIFIED | Exact reverse; best-effort lossiness documented |
| `internal/core/domain/activity/activity.go` | Full P-007 field set, ActivityKind string type, DTOs, sentinels | ✓ VERIFIED | 123+ lines; `ErrActivityNotLoggable` + `ErrActivityCycle` (ADR-BE-001 pattern) |
| `internal/core/ports/activity_repository.go` | Single port: CRUD, ListChildren, GetAncestry, ResolveCommercialContext, ResolveBillability, KindExists, ListKinds, guards | ✓ VERIFIED | Complete interface |
| `internal/adapters/secondary/postgres/activity_repository.go` | Full adapter with recursive CTEs | ✓ VERIFIED | 658+ lines; GetAncestry CTE (line 187), commercial/billability CTEs, subtree delete guards |
| `internal/core/services/activity/activity.go` | ActivityService CRUD + D-2/D-3 validation + `validateParent` + sentinel-guarded Delete | ✓ VERIFIED | 218 lines; validateParent (124-145) shared by Create (78) and Update (106); 9 test functions in activity_test.go |
| `internal/adapters/primary/http/activity_handler.go` | 7 endpoints; detail composes ancestry + commercial context + billable; ErrActivityCycle → 400 | ✓ VERIFIED | Update switch maps ErrActivityCycle → 400 "activity parent would create a cycle" (291-292); boundary unit tests + TestActivityHandlerIntegration |
| `cmd/server/main.go` | Router rewired to activity endpoint set | ✓ VERIFIED | 7 activity routes; no project/subproject routes |
| `internal/core/services/working_group/working_group_integration_test.go` | Re-seeded seedWGData on activities schema | ✓ VERIFIED | Seeds activity_kinds('engagement') + root activities row; returns (orgID, managerID, activityID); zero references to projects/subprojects; 4/4 subtests PASS (re-ran) |
| `internal/adapters/secondary/postgres/activity_ontology_migration_test.go` | Variadic applyMigrations; 013 interleaved; corrected kind assertions | ✓ VERIFIED | phase 6 / task 0 asserted in both up passes; down-reversal proof; pre-state skips 013 (42P01 avoidance) |
| `internal/adapters/secondary/postgres/staffing_schema_migration_test.go` | Call site updated to variadic signature | ✓ VERIFIED | `applyMigrations(t, pool, true, "012_staffing_schema.up.sql")`; cycle test PASS |
| `internal/core/services/testdata/mocks.go` | MockActivityRepo.GetAncestry map-walk + GetAncestryFn override | ✓ VERIFIED | Chain includes starting node (self-parent detection); override honored (line 599-600) |
| Deleted files | project/subproject domain, ports, adapters, services, handlers | ✓ VERIFIED | All gone; zero legacy symbols in internal/ cmd/ |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Service Submit | Manager stage | `resolveManagerStage` (activity → WG → manager+delegates) | ✓ WIRED | Both entry services; green in full suite |
| Service Submit | Unit-manager fallback | `resolveUnitManager` upward unit-tree walk | ✓ WIRED | Personal activities (no contract, no WG) route to unit manager |
| Service Submit | ErrActivityNotLoggable | sentinel on commercial-without-WG | ✓ WIRED | Both services return it; both handlers map to 409 |
| Service Update | ErrActivityCycle | `validateParent` → `GetAncestry` walk of proposed parent; reject when chain contains own id | ✓ WIRED | activity.go:106 → 135-142; tested (3 cycle-reject subtests) |
| Service Create | validateParent | `validateParent(ctx, orgID, uuid.Nil, req.ParentID)` — uniform path check on insert | ✓ WIRED | activity.go:78; create-with-parent subtest PASS |
| Handler Update | ErrActivityCycle → 400 | `errors.Is` case before default | ✓ WIRED | activity_handler.go:291-292 — no longer falls through to 500 |
| Handler | ActivityService + repo | `ActivityHandler` holds service + repo port | ✓ WIRED | Detail endpoint composes GetAncestry/ResolveCommercialContext/ResolveBillability |
| Router | Handlers | `mux.HandleFunc` in main.go | ✓ WIRED | All 7 activity endpoints behind middleware.Auth |
| Entry repos | ListPending chain | activity → WG join | ✓ WIRED | Both repos; postgres package green |
| Entry repos | Subtree list filter | `WITH RECURSIVE activity_subtree` | ✓ WIRED | Both repos; matches SPEC "repository resolves the subtree" |
| WG test fixture | activities schema | seedWGData → activity_kinds + activities rows → `SubprojectID` maps to activity_id column | ✓ WIRED | Repo Create maps legacy field to activity_id column (D-5); 4/4 subtests PASS |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| ActivityHandler.Get | ancestry/commercial_context/billable | repo recursive CTEs | Yes — real DB queries | ✓ FLOWING |
| TimeEntry repo Create | activity_name/activity_kind | scalar-subquery RETURNING join | Yes — joined values in one round trip | ✓ FLOWING |
| Entry ListPending | manager approver set | WG row ManagerID+DelegateIDs | Yes — real resolution | ✓ FLOWING |
| Activity Create | kind catalog | `KindExists` EXISTS query | Yes — real catalog check | ✓ FLOWING |
| Activity Update parent | GetAncestry chain | recursive CTE walk (leaf→root) | Yes — real DB walk; drives ErrActivityCycle | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Migration cycle incl. 013 | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration011_ActivityOntology_UpDownUpCycle|TestMigration012_StaffingSchema_UpDownUpCycle' -count=1 -v` | Both PASS (4.11s + 0.32s) — 013 interleaved; phase 6 / task 0 asserted | ✓ PASS |
| Cycle + parent validation | `go test ./internal/core/services/activity/ -run 'TestService_UpdateParentValidation|TestService_Create' -count=1 -v` | PASS — 6 validation subtests + 10 create subtests (incl. path-check-on-insert) | ✓ PASS |
| WG integration re-seed | `go test ./internal/core/services/working_group/ -run TestWorkingGroupIntegration -count=1 -v` | PASS — 4/4 subtests (CreateAndGetByID, ListByOrg, GetNotFound, Delete) | ✓ PASS |
| Full backend suite | `go test ./... -count=1` | 19/19 packages ok — no FAIL lines (cmd/migrate + cmd/server + http + postgres + 12 services + handlers + middleware + models) | ✓ PASS |
| Build + vet | `go build ./...`; `go vet ./...` | Both exit 0 (re-ran) | ✓ PASS |
| Old-route removal | `rg '/projects|/subprojects' cmd/server/main.go` | No matches | ✓ PASS |
| Legacy-symbol removal | `rg 'project_repository|subproject_repository' internal/ cmd/` | 0 matches | ✓ PASS |
| Immutability 011/012/000 | `git diff --stat HEAD -- migrations/011_activity_ontology.up.sql migrations/012_staffing_schema.up.sql migrations/000_full_schema.up.sql` | Empty — applied history untouched | ✓ PASS |
| Frontend quarantine | `git log --oneline <phase-range> -- web/src/` | 0 commits in full phase range incl. gap fixes | ✓ PASS |
| Debt markers | `rg 'TBD|FIXME|XXX'` across all 10 phase-modified files | 0 matches | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — phase plans/summaries declare no probe scripts (`scripts/*/tests/probe-*.sh`); verification is via the Go testcontainers suite, all key tests re-executed here.

### Requirements Coverage

No plan requirement IDs (P-007-D*, P-008-D*, BE-014-R*, P-011-D*) are registered in `.planning/REQUIREMENTS.md` (0 matches — re-verified; it tracks milestone-level IDs like TEST-*/AUTH-*). Per 09-CONTEXT, the ADRs in `hourglass-vault/decisions/` are the canonical spec. Every frontmatter ID across all 8 plans (union: P-007-D1..D8, P-008-D1/D1a/D1b/D2/D4, BE-014-R1..R6, P-011-D6 — matches the phase's declared set exactly) is accounted for against its ADR and code evidence:

| Requirement | Source ADR | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| P-007-D1 | ADR-P-007 | Table/domain word is `activity` | ✓ SATISFIED | activities table + domain type |
| P-007-D2 | ADR-P-007 | Recursive parent_id; kind free catalog label | ✓ SATISFIED | schema + ActivityKind string type; validateParent path check (09-08) |
| P-007-D3 | ADR-P-007 | contract_id nullable; commercial derived via CTE | ✓ SATISFIED | ResolveCommercialContext CTE |
| P-007-D4 | ADR-P-007 | Entries reference one activity_id NOT NULL | ✓ SATISFIED | migration + domain + repos |
| P-007-D5 | ADR-P-007 | WG anchors at any depth via activity_id | ✓ SATISFIED | migration + WG repo + re-seeded test fixture (09-07) |
| P-007-D6 | ADR-P-007 | Big-bang migration + data migration | ✓ SATISFIED | 011 up/down + 013 forward fix + cycle tests |
| P-007-D7 | ADR-P-007 | billable nullable = inherit | ✓ SATISFIED | schema + ResolveBillability |
| P-007-D8 | ADR-P-007 | Personal activities first-class; unit-manager fallback | ✓ SATISFIED | R-2 fallback in both services |
| BE-014-R1 | ADR-BE-014 | Routing from activity_id chain | ✓ SATISFIED | resolveManagerStage both services |
| BE-014-R2 | ADR-BE-014 | Precedence; ErrActivityNotLoggable | ✓ SATISFIED | sentinel + enforcement + 409 |
| BE-014-R3 | ADR-BE-014 | D-11 skip incl. delegates | ✓ SATISFIED | skipToFinance tests green |
| BE-014-R4 | ADR-BE-014 | Unit-subtree visibility | ✓ SATISFIED | repo-side gating preserved |
| BE-014-R5 | ADR-BE-014 | Schema consequences (FK drops, WG, cutoff, IsPeriodLocked key) | ✓ SATISFIED | migration + IsPeriodLocked(org, activity, date) + WG re-seed (09-07) |
| BE-014-R6 | ADR-BE-014 | Repo collapse + ListPending via activity→WG | ✓ SATISFIED | single port/adapter; both repos |
| P-008-D1/D1a/D1b | ADR-P-008 | availability_windows table + CHECKs | ✓ SATISFIED | 012 migration + cycle test PASS |
| P-008-D2 | ADR-P-008 | Membership validity dates | ✓ SATISFIED | 3 nullable DATE columns |
| P-008-D4 | ADR-P-008 | `hr` role accepted by DB | ✓ SATISFIED (DB) | role CHECK extended; Go model lacks RoleHR (advisory — Phase 10 surfaces) |
| P-011-D6 | ADR-P-011 | Backend activity-shaped; frontend rename deferred | ✓ SATISFIED | /activities endpoint set live; frontend untouched |

**Orphaned requirements:** none — all frontmatter IDs mapped above. No REQUIREMENTS.md IDs map to this phase (phase carries ADR decision IDs only).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `internal/core/domain/working_group/working_group.go` | 22/36/45 | Legacy `EnforceUnitTuple`/`SubprojectID` fields survive the schema drop | ⚠️ WARNING | Schema clean; Go domain + HTTP DTO still carry legacy field (documented deferral to Phase 10 — field rename; the WG integration test now passes the activity id through the legacy field name, exercising the real column) |
| `internal/models/models.go` | 11-25 | No `RoleHR` constant; `IsValid()` rejects 'hr' | ⚠️ WARNING | DB role CHECK accepts 'hr' but Go org service would reject it — role unusable via API until Phase 10 surfaces (advisory) |
| `internal/models/models.go` | 130-370 | Legacy `ProjectID`-bearing types (Project, TimeEntry, Expense, ExpenseItem) | ℹ️ INFO | Dead types — referenced only by testdata factories (which now set ActivityID); no live path |
| `internal/adapters/primary/http/export.go` | 301-303 | `queryProjectID` dead helper (never called) | ℹ️ INFO | Cosmetic residue; export repo rewritten to commercial-chain CTE |
| `internal/models/surreal_models.go` | whole file | Unused legacy Surreal-era models with ProjectID | ℹ️ INFO | No importers found in internal/ or cmd/ |

**Review criticals (advisory — NOT must-have failures, from 09-REVIEW.md, routed to follow-up phases):** CR-01 employee-visibility scoping on entry List endpoints, CR-02 receipt upload IDOR (`POST /expenses/{id}/receipt` lacks ownership/org check), CR-03 activity Get/ListChildren not org-scoped (cross-org read). All three share the root cause that the hex rewrite moved authorization to the service/repo layer without fully implementing it there; handler tests exercise only invalid-input paths plus happy-path integration. These are documented findings for the hardening/follow-up phase, explicitly out of must-have scope per the phase boundary — they do not block Phase 09.

**Debt markers:** no TBD/FIXME/XXX in any phase-modified file (re-scanned including all gap-closure files). Frontend build (`bun run build`) fails with TS errors — this is the *expected* quarantined state per SPEC ("known-broken until Phase 10", excluded from pass criteria); no `web/src/` files were modified by this phase.

### Human Verification Required

### 1. Live end-to-end activity flow smoke test (incl. cycle rejection)

**Test:** Run the server against a testcontainers/live PostgreSQL and exercise the full flow: create parent + child activity, list with filters, fetch detail (ancestry + commercial_context + billable), attempt a cycle-creating PUT `/api/activities/{id}` reparent (parent = own descendant or self), log a time entry with `activity_id`, submit, approve as WG manager.
**Expected:** All endpoints respond correctly; the reparent attempt returns HTTP 400 with "activity parent would create a cycle" (not 500); submit on a commercial activity without an anchored WG returns 409 with the ErrActivityNotLoggable message; D-11 skip routes owner-is-approver entries straight to `pending_finance`; WG manager approve transitions to `pending_finance`.
**Why human:** Requires a running server + DB + authenticated session; handler integration tests (TestActivityHandlerIntegration, http package green in full suite) cover the flow programmatically, but the executor's claimed live curl smoke (09-05) is not reproducible from static inspection, and the 09-08 ErrActivityCycle 400 path is only unit-tested at the service layer.

### Gaps Summary

**All three initial-verification gaps are closed and verified by re-running the tests:**

1. ✅ **kind='phase'** — forward migration 013 relabels subproject-derived rows; cycle test asserts engagement 6 / phase 6 / task 0 / internal 1; 011/012/000 immutable.
2. ✅ **`go test ./...` green** — working_group integration test re-seeded onto the activities schema; full suite 19/19 packages PASS.
3. ✅ **Cycle prevention on `activities.parent_id`** — `ErrActivityCycle` sentinel + shared `validateParent` (exists + same-org + GetAncestry walk) enforced on Create and Update; HTTP 400 mapping; 7 new unit tests.

Score: **16/16 truths verified**, build/vet clean, zero debt markers, all requirement IDs accounted for. The ontology rewrite, staffing schema, and activity-shaped API surface are complete and verified in code. The only outstanding item is the live-server human smoke test (not reproducible from static inspection); three advisory review criticals (visibility scoping, receipt IDOR, cross-org detail reads) are documented for follow-up phases and are not must-have failures. Frontend quarantine is intact per SPEC.

---

_Verified: 2026-07-31T20:45:00Z_
_Verifier: the agent (gsd-verifier)_
