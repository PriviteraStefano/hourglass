---
phase: 00-testing-foundation
plan: 05
subsystem: testing
tags: [go, postgres, bug-fix, integration-tests, schema-migration]
requires:
  - phase: 00-04
    provides: Handler integration test rewrite patterns
  - phase: 00-03
    provides: Service integration test files with pre-existing issue documentation
provides:
  - Full green test suite across all 16 internal packages
  - organization_settings table + auto-create trigger
  - Fixed invitation CreatedBy flow (real UUID instead of hardcoded "system")
  - Fixed expense customer_id column, export queries, and export test FK chain
affects:
  - 00-06 (E2E verification can run on clean suite)
  - All future phases (no skipped tests, no pre-existing failures)

tech-stack:
  added: []
  patterns:
    - DB trigger for auto-creating organization_settings on org INSERT
    - Proper FK seeding in repo integration tests

key-files:
  created: []
  modified:
    - migrations/000_full_schema.up.sql
    - internal/adapters/secondary/postgres/export_repository.go
    - internal/adapters/secondary/postgres/export_repository_test.go
    - internal/adapters/secondary/postgres/exported_test_helpers.go
    - internal/adapters/secondary/postgres/organization_repo_test.go
    - internal/adapters/secondary/postgres/subproject_repository.go
    - internal/adapters/secondary/postgres/user_repository_test.go
    - internal/core/domain/invitation/invitation.go
    - internal/core/services/invitation/invitation.go
    - internal/core/services/invitation/invitation_test.go
    - internal/core/services/invitation/invitation_integration_test.go
    - internal/core/services/password_reset/password_reset_integration_test.go
    - internal/core/services/organization/organization_integration_test.go
    - internal/adapters/primary/http/invitation.go

key-decisions:
  - "Added customer_id column to expenses table (production repo already expected it)"
  - "Added organization_settings table + AFTER INSERT trigger (repo expected it, tests expected trigger)"
  - "Made organization_memberships.user_id nullable (InviteMember flow inserts NULL for pending invites)"
  - "Added 'used' to invitations status CHECK constraint (domain model uses 'used' status)"
  - "Fixed export_repository u.name → CONCAT(u.firstname, ' ', u.lastname) (users table has firstname/lastname, not name)"

requirements-completed: [TEST-05]

duration: 23 min
completed: 2026-06-09
---

# Phase 0 Plan 05: Bug Buffer and Fix Summary

**Fixed 12+ documented bugs across test files, schema, and production code — full test suite green with zero skipped tests**

## Performance

- **Duration:** 23 min
- **Started:** 2026-06-09T18:55:00Z
- **Completed:** 2026-06-09T17:18:55Z
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments

- Fixed all 3 Group A trivial test issues (role, float64, subproject update args)
- Fixed all 4 Group B schema mismatches (expense customer_id, org_settings table + trigger, export query columns, export test FK chain)
- Fixed all 3 Group C skipped service integration tests (invitation CreatedBy, password reset replay, org UpdateSettings)
- Added organization_settings table with AFTER INSERT trigger for auto-creation
- Made organization_memberships.user_id nullable to support invite-before-register flow
- Added 'used' to invitations status CHECK constraint to match domain model
- Fixed export_repository queries (u.name → firstname+lastname concatenation)
- Seeded full FK chain in export tests (subproject, unit, working group for time entries)

## Task Commits

1. **Task 1: Group A — Trivial fixes** - `06d585a` (fix)
2. **Task 2: Group B — Schema mismatches** - `a4d1fd9` (fix)
3. **Task 3: Group C — Service integration tests** - `d8c90bf` (fix)
4. **Task 3b: Remaining assertion/schema fixes** - `1aa9a79` (fix)

**Plan metadata:** (to be committed)

## Files Created/Modified

- `migrations/000_full_schema.up.sql` — Added customer_id to expenses, organization_settings table + trigger, made user_id nullable, added 'used' to invitations status CHECK
- `internal/adapters/secondary/postgres/user_repository_test.go` — Fixed role "owner" → "manager" + assertion
- `internal/adapters/secondary/postgres/organization_repo_test.go` — JSON round-trip for FinancialCutoffConfig comparison
- `internal/adapters/secondary/postgres/subproject_repository.go` — Removed extra projectID arg from Update
- `internal/adapters/secondary/postgres/export_repository.go` — Fixed u.name → CONCAT(firstname, lastname)
- `internal/adapters/secondary/postgres/export_repository_test.go` — seedFullFKChain returns full FK chain, all INSERTs include required columns
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — Added organization_settings to teardown list
- `internal/core/domain/invitation/invitation.go` — Added CreatedBy to CreateInvitationRequest
- `internal/core/services/invitation/invitation.go` — Use req.CreatedBy instead of hardcoded "system"
- `internal/core/services/invitation/invitation_test.go` — Added CreatedBy to test request
- `internal/core/services/invitation/invitation_integration_test.go` — Full rewrite, no skips, all tests working
- `internal/core/services/password_reset/password_reset_integration_test.go` — Unskipped VerifyPreventsReplay
- `internal/core/services/organization/organization_integration_test.go` — Unskipped UpdateSettings
- `internal/adapters/primary/http/invitation.go` — Passes user ID from middleware context

## Decisions Made

- **Added customer_id to expenses table:** Production repo code already expected this column; schema was missing it. Minimal fix: add column to migration rather than remove from repo.
- **Added organization_settings table + trigger:** The org management repo expects this table with an auto-created default row. Added AFTER INSERT trigger on organizations for clean auto-creation in tests and production.
- **Made organization_memberships.user_id nullable:** The InviteMember flow intentionally inserts NULL for pending invites (user hasn't registered yet). NOT NULL was a schema oversight.
- **Exported test FK chain:** Export tests needed time_entries with subproject_id/wg_id/unit_id FK targets. Added seedSubproject, seedUnit, seedWG helpers to seedFullFKChain.
- **Used 'used' in invitations status:** Domain model uses InvitationStatusUsed but DB only allowed 'pending'/'accepted'/'expired'. Added 'used' to CHECK constraint rather than changing domain semantics.

## Deviations from Plan

None - plan executed exactly as written. All documented bugs fixed with test verification.

## Issues Encountered

- **organization_memberships.user_id NOT NULL vs InviteMember NULL insert:** Schema constraint prevented invited-by-email flow. Fixed by removing NOT NULL on user_id column.
- **Invitation Accept fails with status CHECK violation:** Domain model uses "used" but DB only allowed "pending/accepted/expired". Added "used" to CHECK constraint.
- **Export test time entries missing required FK columns:** time_entries has subproject_id, wg_id, unit_id all NOT NULL. Extended seedFullFKChain to create the full dependency chain and updated all INSERT statements.

## Known Stubs

None.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundary changes.

## Next Phase Readiness

- Full test suite is green across all 16 internal packages
- No skipped tests remain
- Ready for Plan 06: E2E verification
- Smoke test passes (register → login → authenticated API call)

## Self-Check: PASSED

- All 16 internal packages pass: ✓
- Smoke test (TestSmoke): ✓
- No t.Skip('Plan 05') remaining: ✓
- All 6 commits for plan 00-05 exist: ✓
- All 14 modified files verified on disk: ✓
- SUMMARY.md created: ✓
- STATE.md updated: ✓

---

*Phase: 00-testing-foundation*
*Completed: 2026-06-09*
