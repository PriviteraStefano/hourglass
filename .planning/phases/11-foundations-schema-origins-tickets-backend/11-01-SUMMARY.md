---
phase: 11-foundations-schema-origins-tickets-backend
plan: 01
subsystem: database
tags: [postgres, migrations, tickets, origins, audit-logs, cycle-tests]

# Dependency graph
requires:
  - phase: 10-backend-foundations
    provides: migrations 000-013 (full schema, activity ontology 011, staffing 012) + cycle-test skeleton in the postgres test package
provides:
  - migrations 014-017 (tickets, activity origins, contract sold hours, audit logs) with up/down pairs
  - green 011/012 migration cycle tests (pre-existing red debt closed)
  - extended TeardownTestSchema covering audit_logs / ticket_comments / tickets
  - 4 new up/down/up cycle tests with functional CHECK-semantics guards
affects: [11-foundations-schema-origins-tickets-backend plans 02-06, 12-funding-sources, 13-direction]

# Tech tracking
tech-stack:
  added: [none — pure SQL migrations + Go testing on existing pgx/testify stack]
  patterns:
    - "Three-valued-logic CHECK guard: `discriminator IS NULL OR (<per-type rules with explicit IS [NOT] NULL>)` (D-01, Pitfall 1)"
    - "Additive nullable migrations, no backfill (D-16)"
    - "Up/down/up cycle tests with self-seeded inline pre-state (seed migration fixtures retired)"
    - "FK-ordering via migration numbering (014 before 015 so activities.ticket_id resolves at apply time, A8)"

key-files:
  created:
    - migrations/014_ticket_schema.up.sql / .down.sql
    - migrations/015_activity_origins.up.sql / .down.sql
    - migrations/016_contract_sold_hours.up.sql / .down.sql
    - migrations/017_audit_logs.up.sql / .down.sql
    - internal/adapters/secondary/postgres/ontology_extension_migrations_test.go
  modified:
    - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
    - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
    - internal/adapters/secondary/postgres/exported_test_helpers.go

key-decisions:
  - "Cycle tests self-seed their pre-state inline (helpers + direct SQL) instead of relying on a seed migration fixture — 003_seed.up.sql is retired and scripts/seed_demo.sql is demo data, not a test fixture"
  - "011 test pre-state org MUST use the fixed MVP seed UUID 019df8b0-0001-7000-8000-000000000001 because migration 011 seeds activity_kinds only for that org (catalog FK)"
  - "014 numbered one slot before 015 so the activities.ticket_id FK resolves at apply time (A8 ordering problem solved by numbering)"
  - "CHECK constraints follow `origin_type IS NULL OR (...)` / `contract_type IS NULL OR (...)` — legacy rows with NULL discriminators pass every new CHECK (D-01/Pitfall 1 regression guards proven functionally)"
  - "reviewed_by deliberately unconstrained on employee_proposal origins (D-02, research OQ1)"

patterns-established:
  - "Pre-state skip lists in older cycle tests must include newer migrations that depend on their skipped schema level (015 ALTERs activities, which only exists after 011)"
  - "Constraint-violation assertions use pgconn.PgError (SQLSTATE 23514 + constraint name), not bare require.Error"

requirements-completed: [FND-01, FND-03, TICK-01, TICK-02, TICK-04, TICK-05, FND-04]

# Metrics
duration: 6min
completed: 2026-08-07
---

# Phase 11 Plan 1: Migration Foundation (014-017) + Cycle-Test Debt Fix Summary

**Four additive migrations (tickets, activity origins, contract sold hours, audit logs) with three-valued-logic CHECK guards that legacy rows pass, plus the pre-existing red 011/012 cycle tests fixed by self-seeding their pre-state and a teardown list extended for the new tables**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-07T09:48:08Z
- **Completed:** 2026-08-07T09:54:05Z
- **Tasks:** 3
- **Files modified:** 11 (8 created, 3 modified)

## Accomplishments

- **Pre-existing red debt closed first (research Pitfall 3):** `TestMigration011_ActivityOntology_UpDownUpCycle` and `TestMigration012_StaffingSchema_UpDownUpCycle` failed because the historical `003_seed.up.sql` fixture is no longer a migration (seed data moved to `scripts/seed_demo.sql`, which `applyMigrations` never loads). Both tests now self-seed exactly the rows their assertions require — 011 seeds the two-level pre-state (fixed seed org UUID, 6 projects / 6 subprojects / 6 WGs / 12 time entries / 6 expenses with 4 project-linked + 2 NULL), 012 seeds 6 memberships. Assertions and migrations untouched (ADR-BE-004).
- **Migrations 014-017** with up/down pairs: tickets + ticket_comments (kind/status CHECK vocabulary, nullable `dismissed_hours` for TICK-04, FK indexes), activities origin discriminator + 5 ref columns with `activities_origin_refs_check` (three-valued-logic guard, reviewed_by unconstrained per D-02), contracts contract_type/sold_hours/sold_period with `contracts_sold_check` (D-08/D-09), general append-only audit_logs with JSONB payload + entity index (D-05). Numbered 014 before 015 so the tickets FK resolves at apply time (A8).
- **Four new up/down/up cycle tests** mirroring the 012 skeleton with functional Pitfall-1 guards: legacy all-NULL rows pass every new CHECK, valid per-type rows pass, mixed-refs and support-without-period rows fail with SQLSTATE 23514 on the named constraint; JSONB payload round-trips intact.
- **Teardown list extended** with `audit_logs` / `ticket_comments` / `tickets` before `activities` (dependency-ordered shape, Pitfall 8).
- **Full suite green:** `go test ./...` passes in every package including the previously red 011/012 tests.

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix pre-existing red migration cycle tests + extend teardown list** - `07fbdbc` (fix)
2. **Task 2: Create migrations 014-017 (up/down pairs)** - `e21c0c7` (feat)
3. **Task 3: Write migration cycle tests 014-017** - `579e113` (test)

## Files Created/Modified

- `migrations/014_ticket_schema.up.sql` / `.down.sql` - tickets + ticket_comments tables, kind/status CHECK vocabulary, CASCADE comment delete, FK indexes; down drops both tables
- `migrations/015_activity_origins.up.sql` / `.down.sql` - activities origin_type + 5 nullable ref columns, `activities_origin_type_check`, `activities_origin_refs_check` (three-valued-logic guard), idx_activities_ticket_id; down reverses constraints/columns/index
- `migrations/016_contract_sold_hours.up.sql` / `.down.sql` - contracts contract_type/sold_hours/sold_period + `contracts_sold_check`; down reverses
- `migrations/017_audit_logs.up.sql` / `.down.sql` - append-only audit_logs + idx_audit_logs_entity; down drops table
- `internal/adapters/secondary/postgres/ontology_extension_migrations_test.go` - four cycle tests (014-017) with functional CHECK-semantics assertions
- `internal/adapters/secondary/postgres/activity_ontology_migration_test.go` - self-seeded two-level pre-state via new `seedLegacyMVPPrestate` helper; skip list extended with 014-017
- `internal/adapters/secondary/postgres/staffing_schema_migration_test.go` - self-seeded 6 memberships; skip list extended with 014-017
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - teardown drop list extended with audit_logs/ticket_comments/tickets

## Decisions Made

- Cycle tests self-seed pre-state inline (helpers where they cover the table, direct SQL otherwise) — the retired seed fixture is not restored and demo data is not loaded into tests.
- The 011 pre-state org reuses the historical MVP seed UUID because migration 011's kind-catalog seed targets that exact org; any other org id would fail the `(org_id, kind)` catalog FK.
- `seedOrgID` is a `uuid.UUID` (parsed via `uuid.MustParse`) so it can be passed to both `seedUnit` and SQL parameters.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 011/012 pre-state skip lists extended with 014-017**
- **Found during:** Task 3 (whole-package test run)
- **Issue:** Once migrations 014-017 landed, `applyMigrations` in the 011 test applied 015's `ALTER TABLE activities` against a pre-state where 011 itself is skipped — `relation "activities" does not exist (SQLSTATE 42P01)`. The 012 pre-state would silently absorb 014-017, drifting from its intended 000-013 level.
- **Fix:** Added the four new migration files to both tests' skip lists so pre-states stay exactly at 000-010 (011 test) and 000-013 (012 test) per ADR-BE-004.
- **Files modified:** activity_ontology_migration_test.go, staffing_schema_migration_test.go
- **Verification:** `go test ./internal/adapters/secondary/postgres/ -count=1` green
- **Committed in:** 579e113 (Task 3 commit)

**2. [Rule 1 - Bug] seedOrgID constant was an untyped string, seedUnit expects uuid.UUID**
- **Found during:** Task 1 (first compile)
- **Issue:** `cannot use seedOrgID (untyped string constant) as uuid.UUID value in argument to seedUnit`.
- **Fix:** Declared `seedOrgID` as `uuid.MustParse("019df8b0-0001-7000-8000-000000000001")` — usable both as SQL parameter and in helper arguments.
- **Files modified:** activity_ontology_migration_test.go
- **Verification:** tests compile and pass
- **Committed in:** 07fbdbc (Task 1 commit)

**3. [Rule 1 - Bug] 014 CHECK vocabulary formatting did not match the plan's literal grep pattern**
- **Found during:** Task 2 verification
- **Issue:** The kind/status CHECKs were written with spaces after commas (`'question', 'bug'`); the acceptance criteria grep expects the exact pattern `CHECK (kind IN ('question','bug','change','evolution'))`.
- **Fix:** Removed the spaces to match the plan's exact vocabulary strings verbatim.
- **Files modified:** migrations/014_ticket_schema.up.sql
- **Verification:** acceptance-criteria greps all pass
- **Committed in:** e21c0c7 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs, 1 Rule 3 blocking)
**Impact on plan:** All three fixes were necessary for correctness (skip-list drift would break the 011 cycle test; the UUID type fix is a compile requirement; the CHECK formatting matches the plan's pinned vocabulary). No scope creep — the schema shapes, assertions, and expected values are exactly as planned.

## Issues Encountered

None beyond the deviations above — all resolved within their tasks. The red 011/012 tests failed exactly as research Pitfall 3 predicted, and the fix strategy (self-seeding) worked first try.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Migrations 014-017 are applied by every test pre-state and by `cmd/migrate` (sorted order), so plans 02-06 (sold_hours, origins, tickets, audit code plans) compile and test against these exact tables.
- The migration cycle tests double as the Pitfall-1 regression guards: any future CHECK added without the three-valued-logic guard will fail the legacy-row functional assertions.
- Pre-existing 011/012 cycle-test debt is closed — no red tests remain in the postgres package.
- Next plan: ready for 11-02 (sold_hours / origins / tickets code plans).

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
