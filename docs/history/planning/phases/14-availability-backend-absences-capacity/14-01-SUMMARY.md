---
phase: 14-availability-backend-absences-capacity
plan: 01
subsystem: database
tags: [postgres, migrations, check-constraints, availability, work-schedules, certificates]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane
    provides: migration 021 named-constraint + 2VL never-NULL-satisfiable CHECK precedent, 022 cycle-test template, TestPool/readMigration/applyMigrations/seed helpers
provides:
  - Migration 023: availability_windows status vocabulary extended to ('declared','confirmed','rejected','withdrawn') + rejection_reason with 2VL guard
  - Migration 024: contract_types work-schedule template table + organization_memberships.contract_type_id/day_hours_override
  - Migration 025: certificate_attachments BYTEA document store with closed entity_type CHECK
  - Teardown drop order + seedAvailabilityWindowWithCert/seedContractType helpers for later availability suites
affects: [14-02 status vocabulary constants, 14-05 attach mutator, 14-06 schedule resolution, 14-07/14-08 capacity reads]

actuals:
  tokens: 8020    # chars/4 over realized diff (32079 chars, 9 files)
  tasks: 3
  commits: 7

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Named CHECK constraints probed by SQLSTATE 23514 + constraint name (021 precedent)"
    - "2VL never-NULL-satisfiable CHECK form for mandatory-reason guards (status <> 'rejected' OR reason IS NOT NULL)"
    - "Up/down/up cycle tests skipping the migration under test in pre-state (ADR-BE-004)"
    - "Down migrations downgrade unrepresentable rows before restoring a CHECK (least-privilege default)"

key-files:
  created:
    - migrations/023_availability_status.up.sql
    - migrations/023_availability_status.down.sql
    - migrations/024_work_schedules.up.sql
    - migrations/024_work_schedules.down.sql
    - migrations/025_certificate_attachments.up.sql
    - migrations/025_certificate_attachments.down.sql
    - internal/adapters/secondary/postgres/availability_ontology_migrations_test.go
  modified:
    - internal/adapters/secondary/postgres/exported_test_helpers.go
    - internal/adapters/secondary/postgres/staffing_schema_migration_test.go

key-decisions:
  - "024 carries NO default-flag column on contract_types: the org default schedule is an org_settings key (D-14-18, research OQ4); header comment avoids the literal token so grep acceptance checks don't trip (Phase 13 'unique' lesson)"
  - "024 day_hours JSONB nullable + validated code-side per the 022 'CHECK on JSONB is infeasible' convention (research OQ5)"
  - "025 size cap enforced at the handler (expense.go MIME/size gate pattern), not in SQL — no BYTEA precedent in repo"
  - "023 down downgrades rejected/withdrawn rows to 'declared' before restoring the two-value CHECK (23514-safe restore)"

patterns-established:
  - "Migration pre-state skip-list maintenance: tests pinning pre-state before a table-creating migration must skip later migrations that reference that table (012 test now skips 023-025)"
  - "Teardown FK-safe drop order: certificate_attachments before availability_windows; contract_types after organization_memberships"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "Migration 023 — extended status CHECK ('declared','confirmed','rejected','withdrawn') + rejection_reason with the 2VL never-NULL-satisfiable guard, proven by 23514 + constraint name, down restores the original two-value CHECK"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_ontology_migrations_test.go#TestMigration023_AvailabilityStatus_UpDownUpCycle"
        status: pass
    human_judgment: false
  - id: D2
    description: "Migration 024 — contract_types table with named cadence/hours CHECKs + nullable day_hours JSONB; organization_memberships.contract_type_id FK + day_hours_override; no default-flag column; down drops columns before table"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_ontology_migrations_test.go#TestMigration024_WorkSchedules_UpDownUpCycle"
        status: pass
    human_judgment: false
  - id: D3
    description: "Migration 025 — certificate_attachments BYTEA store with closed-scope named entity_type CHECK, FK chain, (entity_id) index; teardown drop order + seedAvailabilityWindowWithCert/seedContractType helpers compile and are usable by later suites"
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_ontology_migrations_test.go#TestMigration025_CertificateAttachments_UpDownUpCycle"
        status: pass
      - kind: other
        ref: "go build ./... (seed helpers referenced by later plans are additive-only)"
        status: pass
    human_judgment: false

# Metrics
duration: 90min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 01: Absence Lifecycle Schema Migrations 023-025 Summary

**Migrations 023/024/025 landed append-only with up/down pairs and up/down/up cycle tests: availability_windows gains the full D-14-08 status vocabulary with a 2VL reject-reason guard, contract_types + membership override make the work-schedule model schema-ready, certificate_attachments provides DB-backed BYTEA document storage — plus the FK-safe teardown order and two seed helpers every later availability plan's testcontainers suite depends on.**

## Performance

- **Duration:** ~90 min active (wall span 2026-08-09T19:12Z → 2026-08-11T12:31Z includes machine clock jumps between sessions)
- **Started:** 2026-08-09T19:12:48Z
- **Completed:** 2026-08-11T12:31:18Z
- **Tasks:** 3 (each a full TDD cycle: RED + GREEN)
- **Files modified:** 9 (6 new migrations, 1 new test file, 2 modified)

## Accomplishments

- **Migration 023 (D-14-08/09):** `availability_windows_status_check` drop+recreated under the SAME name with the full vocabulary; `rejection_reason TEXT` + `availability_windows_reject_reason_check` in the never-NULL-satisfiable 2VL form — a rejected row with NULL reason is FALSE OR FALSE (Pitfall 1 guard). Down restores the original two-value CHECK verbatim (downgrading terminal rows first, 23514-safe).
- **Migration 024 (D-14-16/17/18):** `contract_types` template table with NAMED `contract_types_cadence_check` (week|month) + `contract_types_hours_period_check` (> 0), nullable `day_hours` JSONB validated code-side (022 convention); `organization_memberships` gains `contract_type_id` FK + `day_hours_override` JSONB. NO default-flag column — org default schedule is an org_settings key (research OQ4).
- **Migration 025 (D-14-07):** `certificate_attachments` — org-scoped FK table (017 form), named closed-scope `certificate_attachments_entity_type_check` (`'availability_window'` only), BYTEA storage, `(entity_id)` access index. Size caps enforced at the handler, not in SQL.
- **Test infrastructure:** `availability_ontology_migrations_test.go` with three up/down/up cycle tests probing every CHECK by 23514 + constraint name; teardown list updated FK-safe (`certificate_attachments` before `availability_windows`, `contract_types` after `organization_memberships`); `seedAvailabilityWindowWithCert` + `seedContractType` helpers added (WithCert naming avoids the Phase 13 same-package collision — checker BLOCKER).

## Task Commits

Each task was committed atomically (TDD: test commit then feat commit):

1. **Task 1: Migration 023 (TDD)** - `a1b70bd` (test: cycle test 023), `707422d` (feat: 023 up/down + 012 skip-list fix)
2. **Task 2: Migration 024 (TDD)** - `48760d6` (test: cycle test 024), `9fb1e9d` (feat: 024 up/down)
3. **Task 3: Migration 025 + teardown/seed helpers (TDD)** - `05f5ab9` (test: cycle test 025), `4fd0120` (feat: 025 up/down + seed helpers)
4. **Cross-task fix:** `d01410b` (fix: teardown drop order — see deviations)

**Plan metadata:** pending (committed after this file)

## Files Created/Modified

- `migrations/023_availability_status.up.sql` - Extended status CHECK (drop+recreate, same name) + rejection_reason + 2VL guard
- `migrations/023_availability_status.down.sql` - Drop guard/column, downgrade terminal rows, restore original two-value CHECK
- `migrations/024_work_schedules.up.sql` - contract_types table (named CHECKs, nullable day_hours) + membership override columns
- `migrations/024_work_schedules.down.sql` - Drop membership columns first, then table (022 precedent)
- `migrations/025_certificate_attachments.up.sql` - BYTEA document store with closed entity_type CHECK + index
- `migrations/025_certificate_attachments.down.sql` - Drop table
- `internal/adapters/secondary/postgres/availability_ontology_migrations_test.go` - TestMigration023/024/025 cycle tests
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - Teardown list entries + seedAvailabilityWindowWithCert/seedContractType
- `internal/adapters/secondary/postgres/staffing_schema_migration_test.go` - 012 test pre-state skip list extended (see deviations)

## Decisions Made

- **No default-flag column on contract_types** — org default schedule is an org_settings key (D-14-18, research OQ4). The 024 header comment deliberately avoids the literal token `is_default` so grep-based acceptance checks don't trip (Phase 13 'unique' comment lesson, STATE.md).
- **day_hours nullable JSONB** — validated code-side per the 022 convention; a CHECK on JSONB content is infeasible by design (research OQ5).
- **025 size caps at handler, not SQL** — no BYTEA precedent in the repo; standard PG BYTEA with the expense.go MIME/size gate pattern transferred.
- **023 down data accommodation** — rejected/withdrawn rows downgraded to 'declared' before the two-value CHECK restore (mirrors 012 down's hr→employee downgrade).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Teardown drop order fix needed mid-plan, not at Task 3**
- **Found during:** Task 2 (after 024 GREEN)
- **Issue:** The shared testcontainers pool persists across tests in one package run. `TestMigration023`'s pre-state applies 024 (it skips only 023), so `contract_types` exists when `TestMigration024` starts — and the teardown list did not yet drop it, so 024's UP failed with SQLSTATE 42P07 "relation contract_types already exists" whenever 023 ran first (exactly how `go test ./...` runs them). Same hazard awaited 025.
- **Fix:** Added `certificate_attachments` (before `availability_windows`) and `contract_types` (after `organization_memberships`) to the TeardownTestSchema list — the exact FK-safe edit the plan scheduled for Task 3, landed early to keep the suite green.
- **Files modified:** internal/adapters/secondary/postgres/exported_test_helpers.go
- **Verification:** `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration02[3-5]' -count=1` green; full package green
- **Committed in:** d01410b (standalone fix commit)

**2. [Rule 3 - Blocking] Pre-existing 012 cycle test broken by 023's reference to availability_windows**
- **Found during:** Task 1 (after 023 GREEN)
- **Issue:** `TestMigration012_StaffingSchema_UpDownUpCycle` pins its pre-state before 012 and skips 014-020, but 021-025 were not in its skip list. Once 023 existed, the pre-state applied it against a schema without `availability_windows` (012 is the migration under test) → 42P01. This violated Task 1's acceptance criterion "existing 011/012 cycle tests still green".
- **Fix:** Extended the test's skip list with 023/024/025 (025 would break it identically via its FK to availability_windows), with an explanatory comment.
- **Files modified:** internal/adapters/secondary/postgres/staffing_schema_migration_test.go
- **Verification:** `go test -run 'TestMigration01[12]'` green
- **Committed in:** 707422d (folded into the 023 feat commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 3 blocking)
**Impact on plan:** Both fixes were required to keep the pre-existing migration cycle suite green once the new migrations landed — the exact outcome the plan's acceptance criteria demanded. No scope creep; the teardown fix was plan-scheduled work moved earlier.

## Issues Encountered

- **OrbStack Docker daemon wedged twice** under sustained testcontainers load (each wedge blocked `docker ps`/`go test` indefinitely; the full-package run hung ~15 min). Resolved by restarting OrbStack + cleaning stale containers; the full postgres package then passed in 42s. Environmental, not code-related — no code changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-02 (status vocabulary constants):** the DB vocabulary ('declared','confirmed','rejected','withdrawn') is now pinned by 023's CHECK — the domain constants plan must match it exactly (no drift, 14-PATTERNS anti-pattern 2).
- **14-05 (attach mutator):** `certificate_attachments` is schema-ready; `seedAvailabilityWindowWithCert` exists for its repo tests.
- **14-06 (schedule resolution):** `contract_types` + `organization_memberships.contract_type_id/day_hours_override` are ready for the fallback chain reads (override → contract_type → org default → 8h Mon–Fri).
- **14-07/14-08 (capacity reads):** teardown drop order is FK-safe; every later availability suite's testcontainers setup is unblocked.
- The Phase 13 `seedAvailabilityWindow` (direction_repository_test.go) remains untouched and coexists with the new WithCert variant.

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*
## Self-Check: PASSED
