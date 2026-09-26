# Deferred Items — Phase 13 (direction-backend-the-plan-plane)

Items discovered during plan execution that are out of the discovering plan's
scope. Each entry names the owning plan/subsystem so a follow-up can be
routed correctly.

| Date | Found in | Item | Owner | Status |
|------|----------|------|-------|--------|
| 2026-08-08 | 13-03 (full-suite verification) | `TestMigration011_ActivityOntology_UpDownUpCycle` fails: migration `021_direction_rows.up.sql` errors `relation "activities" does not exist` (SQLSTATE 42P01). Cause: 13-01 added migrations 021/022 without extending the 011 cycle test's skip list — its pre-state applies 000-010 then globs every remaining `migrations/*.up.sql` including 021, which references `activities` (exists only after 011). Pre-existing since 13-01 (13-01 deferred full-suite to wave merge). Fix per 12-01 precedent (`ae7f4a6`): add `021_direction_rows.up.sql` + `022_org_settings.up.sql` to the skip list in `internal/adapters/secondary/postgres/activity_ontology_migration_test.go`. Verified: only this one test fails; `go build ./...`, `go vet ./...`, and all 13-03 packages + dependents pass. | 13-01 follow-up / wave merge | open |
