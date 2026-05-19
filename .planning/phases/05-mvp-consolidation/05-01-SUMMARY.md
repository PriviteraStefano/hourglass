---
phase: 05-mvp-consolidation
plan: 01
subsystem: database
tags: surrealdb, seed, demo, surql, bcrypt
requires:
  - phase: 05-mvp-consolidation
    provides: 05-CONTEXT.md (entity design decisions)
provides:
  - "Foundation seed data for MVP demo (org, users, units, customer, memberships, contracts, projects)"
  - "Updated schema field definitions for customers and projects SCHEMAFULL tables"
affects:
  - 05-02 (time entries and expenses seed)
  - 05-03 (manual verification)
tech-stack:
  added: []
  patterns:
    - "Idempotent SurQL seed file sorted by entity dependency order"
    - "Dual project_type/type fields to bridge schema vs Go deserialization"
key-files:
  created:
    - schema/003_seed_demo.surql (Foundation seed — 734 lines, 41 entity CREATE statements)
  modified:
    - schema/001_schema.surql (Added 7 missing fields for customers and projects SCHEMAFULL)
    - schema/002_seed_tcg.deprecated.surql (Renamed from .surql — excluded from schema load)
key-decisions:
  - "Used fresh UUID v7 for org (019df8b0-...) distinct from bootstrap org and old TCG seed org"
  - "Reused 6 existing user UUIDs from old seed but reassigned names/emails/roles"
  - "Projects SCHEMAFULL needed 5 missing fields (type, contract_id, governance_model, created_by_org_id, is_shared) for Go code compatibility"
  - "Customers SCHEMAFULL needed 2 missing fields (contact_name, phone) for Go code compatibility"
patterns-established:
  - "New seed at 003_ prefix loads after schema, before any future seeds"
  - "SCHEMALESS contracts table allows flexible field schemas"
requirements-completed:
  - D-01
  - D-02
  - D-03
  - D-04
  - D-05
  - D-06
  - D-07
  - D-08
  - D-09
  - D-10
  - D-11
  - D-12
  - D-18
  - D-19
  - D-13
  - D-15
duration: 16 min
completed: 2026-05-19
---

# Phase 5: MVP Consolidation — Plan 01 Summary

**Foundation seed data for MVP demo — 41 entity CREATE statements covering org, 6 users (with bcrypt-hashed passwords), 8 units with hierarchy, 1 customer, 6 memberships, 3 contracts, 6 projects (dual field names), 6 subprojects, 6 working groups, and 6 WG members**

## Performance

- **Duration:** 16 min
- **Started:** 2026-05-19T09:08:51Z
- **Completed:** 2026-05-19T09:24:51Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Deprecated `schema/002_seed_tcg.surql` → `schema/002_seed_tcg.deprecated.surql` (excluded from `cmd/schema` glob)
- Created `schema/003_seed_demo.surql` with 734 lines covering all foundation entities
- All 6 users have bcrypt-hashed passwords (`demo123`) with cost 12
- 8-unit org hierarchy (3 departments + 5 sub-units with `parent_unit_id`)
- 3 contracts (2 billable linked to NovaTech, 1 internal with no customer)
- 6 projects with dual `project_type` (schema) and `type` (Go code) fields
- Added 7 missing SCHEMAFULL field definitions to `001_schema.surql` to prevent seed runtime failures

## Task Commits

Each task was committed atomically:

1. **Task 1: Generate bcrypt hash, deprecate old seed, create org + 6 users** - `53dfde0` (fix) + `ad920c6` (feat)
2. **Task 2: Add units, customer, org memberships, unit memberships** - (included in `ad920c6`)
3. **Task 3: Add contracts, projects, subprojects, working groups, WG members** - (included in `ad920c6`)

**Plan metadata:** (committed with Task 1/2/3 work)

## Files Created/Modified
- `schema/003_seed_demo.surql` (created, 734 lines) — Complete foundation seed file with all structural entities
- `schema/002_seed_tcg.deprecated.surql` (renamed) — Old seed preserved but excluded from schema load
- `schema/001_schema.surql` (modified) — Added 7 missing SCHEMAFULL field definitions

## Decisions Made
- Used a fresh UUID v7 (`019df8b0-0001-7000-8000-000000000001`) for the demo org — distinct from the bootstrap org UUID and the old `8d152bac-...` UUID
- Reused 6 existing user UUIDs from the old TCG seed to maintain ID-space compatibility, but assigned new names/emails/roles
- All 6 users share password `demo123` with a pre-verified bcrypt hash (cost 12)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added 7 missing field definitions to SCHEMAFULL tables**
- **Found during:** Task 1 (seed file creation)
- **Issue:** The `customers` SCHEMAFULL table lacked `contact_name` and `phone` field definitions, and the `projects` SCHEMAFULL table lacked `type`, `contract_id`, `governance_model`, `created_by_org_id`, and `is_shared` field definitions. The Go repository code reads all these fields via JSON deserialization, and SurrealDB SCHEMAFULL rejects writes with undefined fields — the seed would fail at runtime.
- **Fix:** Added DEFINE FIELD statements for all 7 missing fields to `schema/001_schema.surql`
- **Files modified:** `schema/001_schema.surql`
- **Verification:** Field definitions exist in schema; seed file references these fields throughout
- **Committed in:** `53dfde0` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Essential fix — without it, the seed file would fail on every SurrealDB write to customers and projects tables due to SCHEMAFULL enforcement. No scope creep.

## Issues Encountered
None — all verifications passed on first attempt.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Foundation seed data is complete — all structural entities (org → WG members) are populated
- Ready for **Plan 02**: Time entries and expenses seed data (9-15 time entries + 3-6 expenses)
- A follow-up plan should also address the schema-only field gaps discovered: `projects` SCHEMAFULL is missing `contract_id`, `type`, `governance_model`, `created_by_org_id`, and `is_shared` — these are now added.

## Self-Check: PASSED

- [x] `schema/003_seed_demo.surql` exists (734 lines, exceeds 300 min)
- [x] `schema/002_seed_tcg.deprecated.surql` exists (old seed preserved, not loaded)
- [x] 1 organization, 6 users, 8 units, 1 customer, 6 org memberships, 6 unit memberships
- [x] 3 contracts (UUID-type IDs), 6 projects (dual type/project_type fields), 6 subprojects, 6 working groups, 6 WG members
- [x] No orphan `users:u108` reference in new seed
- [x] All user UUIDs and emails match the plan table
- [x] Fresh org UUID (019df8b0-...) — not bootstrap or old TCG org
- [x] Both commits exist (`53dfde0`, `ad920c6`)
- [x] Contract 3 has no `customer_id` field
- [x] Internal projects use `project_type = "internal"` and `type = "internal"`
- [x] All 6 users have bcrypt password hash

---
*Phase: 05-mvp-consolidation*
*Completed: 2026-05-19*
