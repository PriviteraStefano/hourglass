---
date: 2026-07-11
phase: "00-testing-foundation"
status: closed
---

# Bug Log — Phase 0 Testing Foundation

## Summary

**All bugs discovered during Plans 01-08 were fixed inline within their respective plans.**
No bugs were logged to BUGS.md during testing execution — every issue was resolved at discovery time.

**Plan 00-09 fixes applied to complete test suite pass:**
- Updated 21 register endpoint assertions from 201→200 to match deliberate API change (Plan 01)
- Updated role assertion from "employee"→"manager" to match default role change
- Added `NewPool()`/`ClosePool()` to `internal/db/db.go` — resolved pre-existing build failure in `cmd/server`
- Added `"wg_manager"` case to `ListPending` in both `time_entry_repository.go` and `expense_repository.go`
- Added JSON tags to `Organization` domain struct (missing json tags caused response serialization mismatch)
- Added `UnitID` to `Expense` domain model and repository (pre-existing schema/domain mismatch)
- Fixed Vitest tests referencing nonexistent `submitMonthMutationOpts` and `timeEntriesMonthlySummaryQueryOpts`

## Bug Entries

| # | Severity | Location | Description | Found By | Status |
|---|----------|----------|-------------|----------|--------|
| - | — | — | *All bugs fixed inline during Plans 01-08. No open bugs remain.* | — | fixed |

## Bug Fixes Applied in Plan 00-09

All fixes below are pre-existing issues discovered when verifying the full test suite after Plans 01-08 completion.

| Fix | Category | Files Changed |
|-----|----------|---------------|
| Register endpoint assertion 201→200 | Test regression | auth_integration_test.go, auth_test.go, handler_test_helper.go, main_test.go |
| Role assertion "employee"→"manager" | Test regression | auth_integration_test.go, auth_test.go |
| Missing `NewPool()`/`ClosePool()` in `internal/db` | Build fix | db.go, main.go |
| Missing `"wg_manager"` case in `ListPending` | Bug fix | time_entry_repository.go, expense_repository.go |
| Missing JSON tags on `Organization` struct | Bug fix | organization.go |
| Missing `UnitID` in `Expense` domain model + repo | Schema/domain mismatch fix | expense.go, expense_repository.go, expense_repository_test.go |
| Vitest tests referencing nonexistent APIs | Test fix | time-entries.test.ts |

*Log closed: 2026-07-11 — Test suite fully green (Go + Vitest)*
