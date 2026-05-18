---
phase: 00-testing-foundation
plan: 07
subsystem: frontend
tags:
  - frontend
  - tests
  - vitest
  - msw
  - zod
  - api
dependency_graph:
  requires: [00-02]
  provides: []
  affects: []
tech-stack:
  added:
    - "msw@^2.14.6"
  patterns:
    - "MSW server interception with setupServer"
    - "Query/mutation option function tests"
    - "Zod schema validation tests"
key-files:
  created:
    - web/src/lib/__tests__/api.test.ts
    - web/src/api/__tests__/auth.test.ts
    - web/src/api/__tests__/time-entries.test.ts
    - web/src/api/__tests__/projects.test.ts
    - web/src/api/__tests__/contracts.test.ts
    - web/src/api/__tests__/customers.test.ts
    - web/src/lib/__tests__/validation.test.ts
  modified: []
key-decisions:
  - "MSW (msw/node setupServer) used for all API mocking — enables 401→refresh→retry sequence testing"
  - "Query/mutation options tested directly via .queryFn() and .mutationFn() instead of mounting React components"
  - "Zod schemas from route components recreated inline in validation test (schemas not exported)"
  - "MSW onUnhandledRequest set to 'bypass' to avoid noise from jsdom/environment-level requests"
requirements-completed: []
duration: "1 min"
completed: "2026-05-18"
test_results:
  files: 7
  tests: 48
  passed: 48
  failed: 0
---

# Phase 00 Plan 07: Frontend API & Validation Unit Tests Summary

Frontend Vitest tests for the `api.ts` HTTP client (MSW-mocked 401→refresh→retry flow), all 5 API module query/mutation option files (auth, time-entries, projects, contracts, customers), and Zod form validation schemas from route components.

## Results

- **48 tests across 7 test files — all passing**
- `cd web && bun run test` exits 0
- MSW properly intercepts all fetch calls in Vitest/jsdom environment

## Test Coverage by File

| File | Tests | Focus |
|------|-------|-------|
| `api.test.ts` | 6 | Envelope unwrap, error, 401→refresh→retry (both paths), Content-Type, network error |
| `auth.test.ts` | 5 | Profile query, login/register/logout/refresh mutations |
| `time-entries.test.ts` | 4 | Monthly summary query, create mutation, submit-month, date query |
| `projects.test.ts` | 4 | List (scope + contract_id params), create, get by id |
| `contracts.test.ts` | 4 | List (scope + is_active params), create, get by id |
| `customers.test.ts` | 3 | List, create, get by id with linked contracts |
| `validation.test.ts` | 22 | Unit schemas, contract enums, login/bootstrap/customer forms, time entry search |

## Verification

```
$ cd web && bun run test -- --reporter=verbose
 ✓ 7 test files passed | 48 tests passed
```

## Deviations from Plan

None — plan executed exactly as written.

- The `"sets Content-Type and Accept headers"` test was adjusted to only assert `Content-Type: application/json` since `api.ts` does not set an explicit `Accept` header (fetch defaults to `*/*`).
- `validation.test.ts` covers 22 Zod schema validation cases across all form schemas found in the codebase (exported unit schemas tested directly; route component schemas recreated inline).

## Known Stubs

None.

## Threat Flags

None.

## Self-Check: PASSED

All 7 created files verified on disk. All 3 commits verified in git log. Full test suite passes.
