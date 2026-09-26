---
phase: 12-coverage-backend-the-allocation-loop
plan: 01
subsystem: database
tags: [postgres, migrations, coverage, allocation-ledger, adr-be-004, cycle-tests]

# Dependency graph
requires:
  - phase: 11-foundations-schema-origins-tickets-backend
    provides: 3VL CHECK house rule (015/016), audit_logs (017), cycle-test self-seed pattern
provides:
  - "activities.beneficiary_unit_id (nullable, 018) — absorption default source (COV-05)"
  - "coverage_allocations tagged-union ledger with 3VL source_check + mandatory-field/vocabulary CHECKs (019)"
  - "coverage_period_closes + coverage_snapshot_rows append-only snapshot tables (020)"
  - "TeardownTestSchema entries + extended pre-existing cycle-test skip lists"
affects: [12-02, 12-03, 12-04, 12-05, 12-06, 12-07, phase-17-surfaces]

# Actuals (#2632) — pairs with the plan's `estimate` (20000) on the same scale.
actuals:
  tokens: 8263    # chars/4 over the realized diff (33052 chars / 4)
  tasks: 2        # tasks completed
  commits: 3      # commits made

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tagged-union allocation ledger: discriminator + nullable refs + 3VL refs-to-type CHECK (015 precedent, D-01)"
    - "Mandatory-field CHECK form `source_type <> 'x' OR col IS NOT NULL` — never NULL-satisfiable (Pitfall 2)"
    - "Entry-level snapshot rows with ON DELETE CASCADE from close header; no aggregates (D-11)"
    - "applyMigrations skip lists must extend when new migrations land (Pitfall 8)"

key-files:
  created:
    - migrations/018_activity_beneficiary_unit.up.sql
    - migrations/018_activity_beneficiary_unit.down.sql
    - migrations/019_coverage_allocations.up.sql
    - migrations/019_coverage_allocations.down.sql
    - migrations/020_coverage_snapshots.up.sql
    - migrations/020_coverage_snapshots.down.sql
    - internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go
  modified:
    - internal/adapters/secondary/postgres/exported_test_helpers.go
    - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
    - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
    - internal/adapters/secondary/postgres/ontology_extension_migrations_test.go

key-decisions:
  - "source_type stays nullable (no NOT NULL): enforcement is the 3VL guard `source_type IS NULL OR (...)` so legacy all-NULL rows pass — mirrors 015 origin_type / 016 contract_type exactly"
  - "entry_id deliberately has no FK (polymorphic D-K); entry_type CHECK ('time') is the schema side of the costed belt-and-braces pair with the 12-04 service branch"
  - "020 has no UNIQUE(org_id, period_start, period_end): duplicate-close rejection is a repo-level in-tx 409 (A6), not a DB unique error"

patterns-established:
  - "Cycle tests skip the migration under test AND every later migration (extended in pre-existing tests when 018-020 landed — Pitfall 8)"
  - "Functional 3VL assertions prove CHECK behavior by failing inserts (pgErr.Code 23514 + constraint name), not by DDL text"

requirements-completed: [COV-01, COV-02, COV-04, COV-05]

coverage:
  - id: D1
    description: "Migration 018 — activities.beneficiary_unit_id nullable FK + idx_activities_beneficiary_unit_id, up/down cycle clean, NULL default valid for legacy rows"
    requirement: COV-05
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go#TestMigration018_ActivityBeneficiaryUnit_UpDownUpCycle"
        status: pass
    human_judgment: false
  - id: D2
    description: "Migration 019 — coverage_allocations ledger with 3VL source_check, source_type/reason/entry_type vocabulary CHECKs, mandatory reason/justification CHECKs; absorption-without-unit and transfer-without-justification rejected (23514), all-NULL legacy row passes, hours>0 enforced"
    requirement: COV-02
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go#TestMigration019_CoverageAllocations_UpDownUpCycle"
        status: pass
    human_judgment: false
  - id: D3
    description: "Migration 020 — coverage_period_closes + coverage_snapshot_rows (entry-level, no aggregates), both indexes, ON DELETE CASCADE from close header proven functionally"
    requirement: COV-04
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go#TestMigration020_CoverageSnapshots_UpDownUpCycle"
        status: pass
    human_judgment: false
  - id: D4
    description: "TeardownTestSchema drops the three coverage tables in dependency order before time_entries; full suite stays green with the new migrations applied everywhere"
    requirement: COV-01
    verification:
      - kind: integration
        ref: "make test (full suite, all packages ok)"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-08-08
status: complete
---

# Phase 12 Plan 01: Coverage Schema Foundation — Allocation Ledger & Snapshots Summary

**Three append-only migrations (018 beneficiary unit, 019 allocation ledger, 020 period-close snapshots) with up/down pairs, 3VL-guarded tagged-union CHECKs proven by failing-insert cycle tests, and a teardown/skip-list regression fix keeping the full suite green**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-08T08:52:14Z
- **Completed:** 2026-08-08T08:58:19Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Migration 018: `activities.beneficiary_unit_id UUID REFERENCES units(id)` (nullable, no 3VL CHECK — mirrors `contract_id`) + `idx_activities_beneficiary_unit_id`; down drops index then column (016 down shape)
- Migration 019: `coverage_allocations` tagged-union ledger (D-01) with the six named constraints — `coverage_allocations_source_check` (3VL refs-to-type guard), `source_type_check` (3-value vocabulary), `reason_vocab_check` (WarrantyBug/UnderEstimate/Goodwill per COV-02), `entry_type_check` ('time' per D-K), `reason_check` + `justification_check` (never NULL-satisfiable mandatory fields) — plus `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)` matching `time_entries.hours` exactly and four access-path indexes; `entry_id` has no FK (polymorphic, service validates)
- Migration 020: `coverage_period_closes` + `coverage_snapshot_rows` (entry-level rows only, no aggregate columns per D-11) with `ON DELETE CASCADE` from the close header, `idx_coverage_snapshot_rows_close/entry`, and no UNIQUE period constraint (409 is repo-level)
- `TeardownTestSchema` gains the three coverage tables before `time_entries` in dependency order (Pitfall 8)
- Cycle tests `TestMigration018/019/020` self-seed pre-state inline (no `seed_demo.sql`) and prove the 3VL guard + mandatory-field CHECKs by failing inserts with `pgErr.Code "23514"` + constraint name — the T-12-01/T-12-02 mitigations asserted functionally, not by DDL text

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrations 018-020 (up/down pairs)** - `66c5678` (feat)
2. **Task 2: Cycle tests + teardown entries** - `d2a7631` (test)
3. **Deviation fix: extend pre-existing cycle-test skip lists** - `ae7f4a6` (fix)

**Plan metadata:** (pending final metadata commit)

## Files Created/Modified

- `migrations/018_activity_beneficiary_unit.up.sql` - nullable `beneficiary_unit_id` + index (COV-05)
- `migrations/018_activity_beneficiary_unit.down.sql` - drop index then column (016 down shape)
- `migrations/019_coverage_allocations.up.sql` - allocation ledger + 6 constraints + 4 indexes (D-01, COV-01/02/05)
- `migrations/019_coverage_allocations.down.sql` - `DROP TABLE IF EXISTS ... CASCADE`
- `migrations/020_coverage_snapshots.up.sql` - close header + entry-level snapshot rows (D-10/D-11)
- `migrations/020_coverage_snapshots.down.sql` - rows then header, CASCADE
- `internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go` - TestMigration018/019/020 cycle tests
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - teardown list += 3 coverage tables before `time_entries`
- `internal/adapters/secondary/postgres/activity_ontology_migration_test.go` - 011 skip list += 018-020
- `internal/adapters/secondary/postgres/staffing_schema_migration_test.go` - 012 skip list += 018-020
- `internal/adapters/secondary/postgres/ontology_extension_migrations_test.go` - 014/015/016/017 skip lists += 018-020

## Decisions Made

- **`source_type` nullable, enforced by the 3VL guard only** — a NOT NULL clause would reject legacy rows; mirrors the 015 origin_type / 016 contract_type precedent exactly (must_have truth, COV-05)
- **`entry_id` has no FK** (D-K polymorphic entry) — the `entry_type IN ('time')` CHECK is the schema side of the costed belt-and-braces pair with the 12-04 service branch
- **No UNIQUE period constraint on 020** — duplicate-close rejection is a repo-level in-tx check returning 409 (A6), per the plan's explicit shape
- **Absorption reasons: exactly `WarrantyBug`/`UnderEstimate`/`Goodwill`** (COV-02) — the Part-5 "plain internal" fourth value is superseded

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing cycle tests broke after migrations 018-020 landed**
- **Found during:** Plan-level verification (`make test`) after Task 2
- **Issue:** `applyMigrations` globs every `*.up.sql` in sorted order and only skips the named files. The pre-existing cycle tests (011/012/014/015/016/017) pin their pre-state "at 000-N" via skip lists that predate 018-020 — so 018 (ALTER activities) applied during 011's pre-state before 011 created `activities`, failing with `SQLSTATE 42P01`. A regression introduced by Task 1's new migration files.
- **Fix:** Extended all six skip lists with `018_activity_beneficiary_unit.up.sql`, `019_coverage_allocations.up.sql`, `020_coverage_snapshots.up.sql` — the same extension Phase 11 applied when 014-017 landed (the tests' comments already anticipated the pattern).
- **Files modified:** `activity_ontology_migration_test.go` (011), `staffing_schema_migration_test.go` (012), `ontology_extension_migrations_test.go` (014/015/016/017)
- **Verification:** `make test` — all packages ok, postgres package uncached run green (27.5s)
- **Committed in:** `ae7f4a6` (dedicated fix commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary for suite correctness — without it the full suite fails. No scope creep; the fix is the mechanical skip-list extension the house pattern prescribes.

## Issues Encountered

None beyond the auto-fixed skip-list regression above. The Task 1 verification command (`-run 'TestMigration018|...'`) targets Task 2's tests, as the plan's done-criteria anticipated; the cycle tests passed on first run (3.676s).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Schema foundation for the coverage plane is verified and green: every later coverage task (repo 12-02/12-03, service 12-04/12-05, handler 12-06/12-07) compiles and tests against the 3VL-guarded tables
- The 12-04 service layer must honor the D-K pair: service branch rejecting `entry_type != 'time'` on top of the schema CHECK
- TestMigration019's failing-insert assertions are the regression guards T-12-01/T-12-02 rely on — keep them green as repo/service plans land

---

*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*

## Self-Check: PASSED
- All 8 key files exist on disk (6 migrations + test file + SUMMARY)
- All 3 commits present in git history (66c5678, d2a7631, ae7f4a6)
- Targeted cycle tests pass (TestMigration018/019/020)
- Full suite green (make test)
