# Phase 0: Testing foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-08
**Phase:** 0-Testing foundation
**Areas discussed:** Plan ordering, Test architecture, Mock tests fate, Bug loop mechanics, Auth scope

---

## Plan Ordering

| Option | Description | Selected |
|--------|-------------|----------|
| Testcontainers first | Setup testcontainers infrastructure first (02→01). Auth bugs fixed after tests exist to verify them. | ✓ |
| Bug fixes first | Keep ROADMAP order — fix auth bugs immediately, set up testcontainers after. | |
| Test-driven | Write failing testcontainer-backed test, then fix, watch test pass. | ✓ |
| Fix and verify | Fix bug first, then write passing test. | |
| Keep separate | Clear separation: 01-Setup testcontainers, 02-Fix auth bugs. | ✓ |
| Merge into one | Single plan: setup + test + fix. | |
| Single fix plan | All 4 auth bugs in one plan. | ✓ |
| One per bug | Separate plan per bug. | |
| Service → Handler → E2E | Rewrite service tests first, then handler, then verify E2E. | ✓ |
| Handler → Service → E2E | Rewrite handler tests first. | |
| Merge into auth fix | Fix known + discovered bugs in one plan. | |
| Keep as buffer plan | Dedicated 05-PLAN for bugs discovered during rewrite. | ✓ |

**User's choice:** Testcontainers first; test-driven; keep separate plans; single auth fix plan; Service→Handler→E2E; keep 05 buffer plan.
**Notes:** All 4 auth bugs in one plan. Test-driven approach (write failing test, then fix).

## Test Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Replace entirely | TestPool spins up testcontainers PostgreSQL. Fully isolated. | ✓ |
| Coexist as alternative | TestPool stays for local dev, testcontainers optional. | |
| Hybrid by layer | Service/handler use testcontainers, repo tests keep TestPool. | |
| Per test function | Each TestFoo gets its own testcontainers instance. | |
| Per package | One testcontainers per _test.go package. | |
| Hybrid | One container per package, each test gets own schema. | ✓ |
| Yes, all tests | Smoke and Playwright both use testcontainers. | ✓ |
| Smoke only | Smoke uses testcontainers, Playwright targets dev server. | |
| None of them | Both need real running PG. | |

**User's choice:** Replace entirely; Hybrid schema-per-test approach; all tests use testcontainers.
**Notes:** Fully self-contained test suite. One container per package, each test gets own schema.

## Mock Tests Fate

| Option | Description | Selected |
|--------|-------------|----------|
| Keep + add integration | Keep all existing mock tests, add new integration tests. | |
| Replace with integration | Convert service tests to use testcontainers instead of mocks. | |
| Hybrid | Keep mocks for pure logic, use testcontainers for repo interaction. | ✓ |
| Fix broken mocks now | Fix mocks that return incorrect values. | ✓ |
| Leave as-is | New tests use testcontainers anyway. | |
| Separate files | auth_test.go (mocks) + auth_integration_test.go (testcontainers). | ✓ |
| Same file | Both mock and integration tests in one file. | |

**User's choice:** Hybrid approach; fix broken mocks now; separate integration test files.
**Notes:** User asked for opinion on file organization — recommended separate files for clear layer distinction.

## Bug Loop Mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Fix immediately | Fix bug right there in the same plan. | |
| Log and batch | Document bugs found during test rewrite, fix in 05-PLAN. | ✓ |
| Skip with TODO | Write test, t.Skip() with reference to logged bug. | |
| Write as comment | Write test code but don't register it. | |
| Mark expected failure | Write test, let it run, capture expected failure. | |

**User's choice:** Log and batch; minor bugs fixed inline, major bugs t.Skip() and human-reviewed in 05-PLAN.
**Notes:** User formalized: minor bugs fixed inline during test rewrite, major bugs logged+t.Skip() and batched to 05-PLAN with human review loop.

## Auth Scope

| Option | Description | Selected |
|--------|-------------|----------|
| 4 known bugs only | Stick to REQUIREMENTS.md scope. | |
| Include refresh rotation | Fix 4 bugs + add refresh token rotation. | |
| All related concerns | Fix 4 bugs plus refresh rotation, cookie name mismatch, all auth concerns. | ✓ |
| Include all auth concerns | Password reset code + rate limiting + refresh rotation + cookie names. | ✓ |
| Core auth only | Refresh rotation + cookie names + 4 bugs. | |

**User's choice:** All related concerns; full auth cleanup including password reset code fix and auth rate limiting.
**Notes:** Clean slate for auth before feature phases begin.

---

## Deferred Ideas

None
