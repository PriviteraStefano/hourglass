# Deferred Items — 09-02 Staffing Schema

Out-of-scope discoveries logged during plan execution (scope boundary: do not
auto-fix issues not caused by the current task's changes).

## 1. 000_full_schema.down.sql missing organization_settings from drop list

- **Discovered:** 2026-07-31, Task 2 verification (live up→down→up cycle)
- **Symptom:** `go run ./cmd/migrate -down -dir migrations` full cascade fails at
  `000_full_schema.down.sql` with
  `pq: cannot drop table organizations because other objects depend on it`
- **Root cause:** `organization_settings` is created in `000_full_schema.up.sql`
  but is absent from the `DROP TABLE` list in `000_full_schema.down.sql`. The
  `organizations` drop is therefore blocked by the `organization_settings`
  FK. Reproduced with only migrations 000–011 present (no 012) — pre-existing,
  unrelated to plan 09-02.
- **Impact:** The full-cascade `-down` CLI is broken regardless of this plan;
  the 012 up/down/up cycle is proven by the testcontainers cycle test
  (`TestMigration012_StaffingSchema_UpDownUpCycle`), which applies 012 up →
  down → up in isolation and passes.
- **Suggested fix (future plan):** add `DROP TABLE IF EXISTS organization_settings;`
  to `000_full_schema.down.sql` before `DROP TABLE IF EXISTS organizations;`
  (also drop `trg_auto_create_org_settings` / `auto_create_org_settings`).
- **Not fixed here:** scope boundary (pre-existing, not caused by 09-02 changes).

## 2. working_group_integration_test.go still seeds dropped projects/subprojects tables

- **Discovered:** 2026-07-31, plan 09-04 verification (`go test ./internal/core/services/...`)
- **Symptom:** `TestWorkingGroupIntegration` fails with `relation "projects" does not exist`
  (SQLSTATE 42P01) — `seedWGData` inserts into `projects`/`subprojects`, both dropped
  by migration 011.
- **Root cause:** migration 011 (09-01) replaced projects/subprojects with `activities`;
  the working_group service integration test was never re-seeded. Pre-existing since
  09-01, documented in 09-03 summary ("working_group_integration_test.go's
  project-seeding test remains runtime-red").
- **Impact:** `go test ./internal/core/services/...` shows exactly one red package
  (working_group). All unit tests are green; contract integration tests pass.
- **Suggested fix (future plan):** re-seed with `activity_kinds` + `activities`
  (the WG's legacy `SubprojectID` field maps to `activities.activity_id`, so the
  service-side call is unchanged) — belongs with the working_group renovation /
  phase 10 field rename, not plan 09-04.
- **Not fixed here:** scope boundary (pre-existing, not caused by 09-04 changes;
  working_group service is absent from plans 04/05 file lists).
