---
phase: 09-activity-ontology
plan: 02
subsystem: database
tags: [postgres, migrations, staffing, availability-windows, adr-p-008, hr-role]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: 011_activity_ontology migration (numbers this migration 012)
  - phase: 00-testing-foundation
    provides: testcontainers migration test infrastructure (SetupTestSchema, TestPool)
provides:
  - `availability_windows` table per ADR-P-008 D-1/D-1a: typed absence kinds (holiday/permit/medical/unavailable), partial-day `hours`, `certificate_ref` (INPS protocol no.), declared/confirmed status
  - `organization_memberships.valid_from` / `valid_until` / `work_permit_expires_at` (D-2, all nullable, open-ended = NULL)
  - `hr` org role accepted by the database (D-4): role CHECK extended
  - index on (org_id, user_id, starts_on, ends_on) for per-person date-range lookups
affects: [09-03 domain+repository collapse, 09-04 service layer, 09-05 http handlers, availability surfacing at assignment time (D-3), payroll export (D-1c)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CHECK extension via drop+recreate (PostgreSQL cannot alter CHECK in place) — same pattern as 004_time_entries_status_check"
    - "Down migration data accommodation: downgrade new-enum rows (hr → employee) before restoring the old CHECK so the constraint restore cannot fail on violating rows"

key-files:
  created:
    - migrations/012_staffing_schema.up.sql
    - migrations/012_staffing_schema.down.sql
    - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
  modified:
    - internal/adapters/secondary/postgres/exported_test_helpers.go

key-decisions:
  - "Migration numbered 012 not 011 — 011 taken by 011_activity_ontology (09-01); per ADR-BE-004 new files continue from the max"
  - "Down migration downgrades existing 'hr' rows to 'employee' before restoring the role CHECK — a rollback after any real hr assignment would otherwise fail with SQLSTATE 23514"
  - "Full-cascade `go run ./cmd/migrate -down` is broken by a pre-existing gap in 000_full_schema.down.sql (organization_settings missing from drop list) — logged to deferred-items.md, out of scope"

patterns-established:
  - "Staffing-schema migration test: testcontainers up/down/up cycle asserting ADR sketch invariants (columns, nullability, CHECKs, index) plus behavioral checks (valid insert OK, invalid kind/inverted dates rejected, hr accepted, bogus rejected)"

requirements-completed: [P-008-D1, P-008-D1a, P-008-D1b, P-008-D2, P-008-D4]

# Metrics
duration: 10min
completed: 2026-07-31
---

# Phase 09 Plan 02: Staffing Schema — Availability Windows + Membership Validity + HR Role Summary

**Additive staffing schema per ADR-P-008 (D-1, D-1a, D-2, D-4): the `availability_windows` table with typed absence kinds, three nullable membership-validity DATE columns, and the `hr` org role added to the role CHECK — one bidirectional 012 migration pair, zero FK coupling to the 011 activity ontology, proven by a testcontainers up/down/up cycle test and live-PostgreSQL behavioral checks.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-07-31T15:44:18Z
- **Completed:** 2026-07-31T15:53:49Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- `availability_windows` matches the ADR-P-008 sketch exactly: `kind` CHECK (`holiday`/`permit`/`medical`/`unavailable`) with `'unavailable'` default, `ends_on >= starts_on` CHECK, nullable `hours` (partial-day permits) and `certificate_ref` (INPS protocol number, never the document), `status` CHECK (`declared`/`confirmed`) with `'declared'` default, org/user/created_by FKs
- `organization_memberships` gains `valid_from` / `valid_until` (NULL = open-ended) / `work_permit_expires_at` (NULL = N/A) — D-2, all nullable
- `hr` role added to the membership role CHECK via drop+recreate (PostgreSQL cannot alter CHECK in place) — D-4
- Composite index on `(org_id, user_id, starts_on, ends_on)` for the per-person date-range lookups the assignment-time surfaces (D-3) will need
- Down migration is an exact reversal with one data accommodation: existing `hr` rows downgrade to `employee` before the old CHECK is restored
- Verified three ways: testcontainers cycle test (up schema invariants → down no-residue → up re-apply), live PostgreSQL 16 container behavioral checks (valid insert OK; invalid kind `'sick'` and inverted dates rejected on CHECK; `role='hr'` accepted; `role='bogus'` rejected), and `go build ./...`

## Task Commits

Each task was committed atomically:

1. **Task 1: Staffing schema up migration (012)** - `f620f04` (feat)
2. **Task 2: Reverse migration + cycle test + teardown extension** - `8028a5b` (feat)
3. **Plan artifact: deferred-items log** - `f249bfc` (docs)

**Plan metadata:** pending docs commit

## Self-Check: PASSED

- All 4 files exist on disk (2 migrations, 1 test, 1 helper)
- All 3 commits present in git log: `f620f04`, `8028a5b`, `f249bfc`
- Self-check re-verified post-write: all 5 SUMMARY-listed files on disk, all 3 commits in git log — PASSED

## Files Created/Modified

- `migrations/012_staffing_schema.up.sql` - Creates availability_windows (ADR-P-008 sketch), adds the three membership validity columns, extends role CHECK with 'hr', adds the org/user/date index
- `migrations/012_staffing_schema.down.sql` - Exact reversal: drops index + table, downgrades any 'hr' rows, restores the original role CHECK, drops the three columns
- `internal/adapters/secondary/postgres/staffing_schema_migration_test.go` - Testcontainers test: pre-state seed, 012 up (schema invariants + hr role + CHECK behavior), 012 down (no residue: table/index/columns gone, hr rejected again), up again (cycle clean)
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - TeardownTestSchema drop-list extended with availability_windows (shared sync.Once container cleanup)

## Decisions Made

- **Migration numbered 012, not 011** — `011` is occupied by `011_activity_ontology` (plan 09-01). The plan text names the files `011_staffing_schema`, but ADR-BE-004 ("new files continue from the max") forces 012. The 09-01 summary explicitly flagged this heads-up.
- **Down migration downgrades existing `hr` rows to `employee`** — the plan's verification sequence itself proves `role='hr'` succeeds post-up; a naive down that just re-adds the old CHECK then fails with SQLSTATE 23514 on the violating row. Downgrading to the least-privilege default (documented in the migration header) makes the rollback genuinely applyable. Schema restoration remains exact — the only accommodation is role value.
- **000_full_schema.down.sql gap is out of scope** — the full-cascade `go run ./cmd/migrate -down` fails at 000 (`organization_settings` missing from its drop list), reproduced without 012 present. Pre-existing; logged to deferred-items.md with a suggested fix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migration renumbered 011 → 012**
- **Found during:** Plan load (before Task 1)
- **Issue:** `migrations/011_activity_ontology.{up,down}.sql` already exist from plan 09-01; writing `011_staffing_schema` would collide with the sequential numbering convention (ADR-BE-004)
- **Fix:** Named the new files `012_staffing_schema.up.sql` / `.down.sql`; every plan reference to "011_staffing_schema" is satisfied by the 012 files
- **Files modified:** migrations/012_staffing_schema.{up,down}.sql
- **Verification:** `go run ./cmd/migrate -up` applies `012_staffing_schema.up.sql` after 011 in sorted order; cycle test reads the 012 files
- **Committed in:** f620f04, 8028a5b

**2. [Rule 1 - Bug] Down migration fails when 'hr' rows exist**
- **Found during:** Task 2 (first test run — `012 down should apply cleanly` failed with SQLSTATE 23514)
- **Issue:** The test's own up-verification assigns `role='hr'`; restoring the old CHECK then fails because a live row violates it. Any real rollback after an hr assignment would hit the same failure.
- **Fix:** Down migration downgrades `role='hr'` rows to `'employee'` (least-privilege default) before dropping the constraint, documented in the migration header
- **Files modified:** migrations/012_staffing_schema.down.sql
- **Verification:** cycle test passes; live `-down` shows the 012 reversal applied cleanly
- **Committed in:** 8028a5b (part of Task 2 commit)

**3. [Rule 2 - Missing Critical] TeardownTestSchema missing availability_windows**
- **Found during:** Task 2 (test infra review)
- **Issue:** The migration creates `availability_windows`, but the shared `sync.Once` testcontainer's teardown drop-list didn't include it — every subsequent test in the package would hit a half-applied 012 state
- **Fix:** Added `availability_windows` to the drop-list (same pattern as 09-01's ontology tables)
- **Files modified:** internal/adapters/secondary/postgres/exported_test_helpers.go
- **Verification:** `TestMigration010`, refresh-token, user, membership-repo, auth, and 012 tests all pass after the change
- **Committed in:** 8028a5b (part of Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking, 1 bug, 1 missing critical)
**Impact on plan:** All auto-fixes were required for the migration to apply at the correct sequence position (012 collision), for the down to be genuinely reversible (hr downgrade), or for the shared test infrastructure to stay sound. No scope creep.

## Issues Encountered

- **Pre-existing `000_full_schema.down.sql` gap (not a deviation, out of scope):** the full-cascade `go run ./cmd/migrate -down` fails at migration 000 with `pq: cannot drop table organizations because other objects depend on it` — `organization_settings` is created in 000-up but absent from 000-down's drop list. Reproduced with only migrations 000–011 (no 012 present), so it predates this plan. The 012 up/down/up cycle itself is proven by the testcontainers cycle test in isolation. Logged to `deferred-items.md` with a suggested fix.

## Verification Results

- `go test ./internal/adapters/secondary/postgres/ -run TestMigration012_StaffingSchema -v` — **PASS** (up applies cleanly, ADR sketch column/CHECK/index invariants, 3 nullable membership columns, hr accepted + bogus rejected, valid insert OK, invalid kind + inverted dates rejected on CHECK, down restores pre-012 state exactly, up→down→up clean)
- `go run ./cmd/migrate -up -dir migrations` against live postgres:16-alpine — **PASS** (applies 012 after 011; `\d availability_windows` matches the sketch; role CHECK contains 'hr'; three columns nullable)
- Live behavioral checks — **PASS** (INSERT holiday window OK; kind 'sick' → `availability_windows_kind_check`; inverted dates → `availability_windows_check`; `UPDATE ... SET role='hr'` → UPDATE 1; `role='bogus'` → `organization_memberships_role_check`)
- `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration010|TestRefreshToken|TestMigration012|TestAuth|TestUser|TestOrganizationMembershipRepo' -count=1` — **PASS** (teardown change regression check)
- `go build ./...` — **PASS**

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Staffing schema foundation in place: availability_windows, membership validity dates, `hr` role at the DB level — all additive, zero coupling to the 011 ontology rewrite
- **Ready for 09-03 (Domain + Repository Collapse)** — the Go model layer will need `RoleHR` added to `internal/models/models.go` `Role` constants / `IsValid()` and `organization_memberships` repository column lists for the three new DATE columns (schema-first landing, code rewrite follows)
- **Blockers/concerns:** none from this plan. `go test ./...` remains red by design until 09-03 rewrites the project/subproject-dependent repositories (accepted mid-phase state from 09-01). The pre-existing 000 down gap is tracked in deferred-items.md for a future plan.

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*
