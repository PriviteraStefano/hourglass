---
phase: 09-activity-ontology
plan: 01
subsystem: database
tags: [postgres, migrations, activity-ontology, adr-p-007, adr-be-014]

# Dependency graph
requires:
  - phase: 00-testing-foundation
    provides: testcontainers migration test infrastructure (SetupTestSchema, TestPool)
  - phase: 08-pre-deployment-hardening-p0-audit-fixes
    provides: migration 010_refresh_token_reuse_detection (numbers the new migration 011)
provides:
  - Recursive `activities` table + `activity_kinds`/`activity_adoptions`/`activity_managers` tables per ADR-P-007
  - `time_entries.activity_id` and `expenses.activity_id` NOT NULL (four-FK capture chain collapsed)
  - working_groups re-anchored to activities; enforce_unit_tuple dropped; budget_caps/financial_cutoff_periods re-pointed
  - Bidirectional migration (011 up/down) with seed data migrated zero-orphan, proven by testcontainers cycle test
affects: [09-02 staffing schema, 09-03 domain+repository collapse, 09-04 service layer, 09-05 http handlers, all project/subproject-dependent Go code]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Same-id migration strategy: activity rows keep the old project/subproject UUIDs so up/down mapping is exact in both directions"
    - "Required-activity fallback: NULL-project expenses get a per-org 'internal' General & Admin activity so activity_id can be NOT NULL"

key-files:
  created:
    - migrations/011_activity_ontology.up.sql
    - migrations/011_activity_ontology.down.sql
    - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
  modified:
    - internal/adapters/secondary/postgres/exported_test_helpers.go

key-decisions:
  - "Migration numbered 011 not 010 — 010_refresh_token_reuse_detection already exists; per ADR-BE-004 new files continue from the max"
  - "budget_caps.project_id rewritten to activity_id (plan omitted it; its FK blocked DROP TABLE projects)"
  - "Same-id migration strategy so the down migration restores the original rows 1:1"
  - "NULL-project expenses (seed: team lunch, office supplies) get a per-org 'internal' General & Admin fallback activity"

patterns-established:
  - "Testcontainers migration verification: apply 000-NNN + seed, run up/down/up cycle, assert schema + data invariants (extends TestMigration010 pattern)"
  - "TeardownTestSchema drop-list must be extended whenever a migration adds tables (shared sync.Once container)"

requirements-completed: [P-007-D1, P-007-D2, P-007-D3, P-007-D4, P-007-D5, P-007-D6, P-007-D7, P-007-D8, BE-014-R5]

# Metrics
duration: 7min
completed: 2026-07-31
---

# Phase 09 Plan 01: Ontology Migration — Activities Schema + Data Migration Summary

**Big-bang schema rewrite replacing projects/subprojects with the recursive `activities` table (ADR-P-007 D-1…D-8, ADR-BE-014 R-5): all six FK rewrites land in one atomic migration, MVP seed data migrates zero-orphan, and a bidirectional 011 up/down pair with a testcontainers cycle test proves up → down → up is clean.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T15:34:11Z
- **Completed:** 2026-07-31T15:40:52Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- `activities` table with the full ADR-P-007 column set: self-ref `parent_id` (ON DELETE RESTRICT), org-level `kind` catalog FK (not a CHECK enum), nullable `contract_id` (ON DELETE RESTRICT), governance CHECK with `creator_controlled` default, tri-state `billable` (NULL = inherit), `budget_amount`
- `activity_kinds` seeded with `engagement`/`phase`/`task`/`internal` for the MVP seed org; `activity_adoptions` + `activity_managers` replace the old project tables
- `time_entries` and `expenses` collapse their FK chains to a single required `activity_id` (D-4); `working_groups` re-anchored to `activity_id` with `enforce_unit_tuple` dropped; `financial_cutoff_periods` and `budget_caps` re-pointed
- Seed data migrated in the same transaction: 6 projects → engagement activities, 6 subprojects → task children, 2 NULL-project expenses → a new 'internal' 'General & Admin' fallback activity — 13 activities total, zero orphans
- Down migration restores the two-level model 1:1 (projects=6, subprojects=6, time_entries fully re-populated, expenses return to their 4/2 project/NULL split) with lossiness documented in the header
- Verified three ways: testcontainers cycle test (up/down/up + schema invariants), real `go run ./cmd/migrate -up|-down|-up` against a live PostgreSQL 16 container with seed data, and full-suite run to enumerate expected mid-phase breakage

## Task Commits

Each task was committed atomically:

1. **Task 1: Schema rewrite (011 up migration)** - `b249460` (feat)
2. **Task 2: Reverse migration + cycle test (011 down)** - `9e04101` (feat)

**Plan metadata:** pending docs commit

## Self-Check: PASSED

- All 5 files exist on disk (2 migrations, 1 test, 1 helper, 1 summary)
- Both task commits present in git log: `b249460`, `9e04101`

## Files Created/Modified

- `migrations/011_activity_ontology.up.sql` - Creates activity_kinds/activities/activity_adoptions/activity_managers, rewrites working_groups/time_entries/expenses/financial_cutoff_periods/budget_caps, migrates seed data, drops projects/subprojects/project_adoptions/project_managers
- `migrations/011_activity_ontology.down.sql` - Restores the two-level model with data, documented lossiness (kind/billable metadata, depth>2 and non-engagement activities)
- `internal/adapters/secondary/postgres/activity_ontology_migration_test.go` - Testcontainers test: pre-state seed, 011 up (schema + zero-orphan + kinds distribution), 011 down (1:1 restore), up again (cycle clean)
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - TeardownTestSchema now drops the 4 new ontology tables (shared-container cleanup)

## Decisions Made

- **Migration numbered 011, not 010** — `010_refresh_token_reuse_detection` (phase 08) already occupies 010. ADR-BE-004: "new files continue from the max". All plan references to `010_activity_ontology` are satisfied as `011_activity_ontology`.
- **budget_caps rewritten like financial_cutoff_periods** (`project_id` → `activity_id`) — the plan's FK list omitted it, but its FK to `projects` would make `DROP TABLE projects` fail. The table is legacy-only (no repository/service usage).
- **Same-id migration strategy** — activity rows keep the old project/subproject UUIDs, making the down migration's reverse mapping exact with no temp mapping tables (only the per-org fallback activities get fresh UUIDs).
- **Per-org 'General & Admin' internal fallback activity** — NULL-project expenses (team lunch, office supplies) cannot stay activity-less under D-4's required `activity_id`; one `internal`-kinded catch-all per org absorbs them.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migration renumbered 010 → 011**
- **Found during:** Plan load (before Task 1)
- **Issue:** `migrations/010_refresh_token_reuse_detection.{up,down}.sql` already exist from phase 08; writing `010_activity_ontology` would collide with the sequential numbering convention (ADR-BE-004)
- **Fix:** Named the new files `011_activity_ontology.up.sql` / `.down.sql`; every plan reference to "010_activity_ontology" is satisfied by the 011 files
- **Files modified:** migrations/011_activity_ontology.{up,down}.sql
- **Verification:** `go run ./cmd/migrate -up` applies `011_activity_ontology.up.sql` after 010 in sorted order
- **Committed in:** b249460, 9e04101

**2. [Rule 3 - Blocking] budget_caps.project_id rewritten → activity_id**
- **Found during:** Task 1 (up migration design)
- **Issue:** The plan's FK rewrite list omits `budget_caps`, whose `project_id UUID REFERENCES projects(id)` would block `DROP TABLE projects` (SQLSTATE 42P01/2BP01 otherwise)
- **Fix:** Dropped the FK, renamed the column to `activity_id`, re-added the FK to `activities`, same pattern as `financial_cutoff_periods`
- **Files modified:** migrations/011_activity_ontology.up.sql, migrations/011_activity_ontology.down.sql
- **Verification:** up applies cleanly; down restores `budget_caps.project_id`; test asserts the column round-trip
- **Committed in:** b249460, 9e04101

**3. [Rule 1 - Bug] `_expense_fallback` dedup broken by volatile uuid**
- **Found during:** Task 1 (first test run)
- **Issue:** `SELECT DISTINCT org_id, gen_random_uuid() ...` — the volatile uuid makes every row distinct, so `DISTINCT` never dedupes by `org_id` and the temp table's PK insert fails (duplicate key, SQLSTATE 23505)
- **Fix:** `SELECT DISTINCT ON (org_id) org_id, gen_random_uuid() ...`
- **Files modified:** migrations/011_activity_ontology.up.sql
- **Verification:** cycle test passes; 13 activities (one fallback per org)
- **Committed in:** b249460 (part of Task 1 commit)

**4. [Rule 1 - Bug] TeardownTestSchema missing the new ontology tables**
- **Found during:** Task 1 (test infra review)
- **Issue:** The migration creates `activities`/`activity_kinds`/`activity_adoptions`/`activity_managers`, but `TeardownTestSchema` (shared `sync.Once` container) didn't drop them — every subsequent test in the package would hit a half-applied 011 state
- **Fix:** Added the four tables to the teardown drop-list
- **Files modified:** internal/adapters/secondary/postgres/exported_test_helpers.go
- **Verification:** existing TestMigration010 + all TestRefreshTokenRepository tests still pass after the change
- **Committed in:** b249460 (part of Task 1 commit)

---

**Total deviations:** 4 auto-fixed (2 blocking, 2 bug)
**Impact on plan:** All auto-fixes were required for the migration to apply at all (010 collision, budget_caps FK) or for the test infra to remain sound. No scope creep — `budget_caps` handling reuses the exact `financial_cutoff_periods` pattern the plan already specified.

## Issues Encountered

- **Expected mid-phase test breakage (not a deviation):** after 011 lands, every test that seeds `projects`/`subprojects` fails with `relation "projects" does not exist` (SQLSTATE 42P01). This is inherent to the big-bang migration landing before the Go code is rewritten — the phase's own sequencing (09-03 Domain + Repository Collapse, 09-04 Service Layer, 09-05 HTTP handlers). Failing packages: `internal/adapters/secondary/postgres`, `internal/adapters/primary/http`, `internal/core/services/{contract,project,working_group}` (6 packages). All non-project-dependent tests (auth, unit, customer, cmd, migration-010, refresh-token, my cycle test) pass. `go build ./...` is green.

## Verification Results

- `go test ./internal/adapters/secondary/postgres/ -run TestMigration011 -v` — **PASS** (up applies cleanly, ADR sketch column/FK/CHECK invariants, zero orphaned entries, kinds distribution 6/6/1, down restores 1:1, up→down→up clean)
- `go run ./cmd/migrate -up -dir migrations` against live postgres:16-alpine — **PASS** (applies 011 after 010; orphan counts 0/0; kinds 6/6/1; old tables 0; enforce_unit_tuple 0)
- `go run ./cmd/migrate -down` then `-up` again — **PASS** (down: projects=6, subprojects=6, time_entries NULL-FKs=0, expenses project-NULL=2; up: activities=13, orphans=0)
- `go build ./...` — **PASS**

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Schema foundation for the whole phase is in place: activities, activity_kinds, activity_adoptions, activity_managers; both entry types pin a required activity_id
- **Ready for 09-02 (Staffing Schema)** — builds on activity_managers
- **Heads-up for the 09-02 executor:** 09-02-PLAN.md names its migration `011_staffing_schema.{up,down}.sql`, but `011` is now taken by `011_activity_ontology` (this plan). Per ADR-BE-004 it must be renumbered **012** (same Rule 3 fix applied here).
- **Blockers/concerns:** the Go layer still compiles but its project/subproject-dependent integration tests fail until 09-03 rewrites the repositories. Until then, `go test ./...` is red by design — this is the accepted mid-phase state, not a regression.

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*
